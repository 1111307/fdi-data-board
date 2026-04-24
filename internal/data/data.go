package data

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"gorm.io/driver/clickhouse"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"fdi_data_board/internal/conf"
	"fdi_data_board/internal/data/orm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewGreeterRepo,
	NewGreeterGrpcRepo,
	NewWebsocketRepo,
	NewQuerySceneRepo,
)

// Data .
type Data struct {
	mysqlDB *gorm.DB
	ckDB    *gorm.DB
	dorisDB *gorm.DB

	rDB *redis.ClusterClient

	anyConn *grpc.ClientConn
}

func (d *Data) RedisDB(ctx context.Context) *redis.ClusterClient {
	return d.rDB
}

func newMysqlDB(c *conf.Data) *gorm.DB {
	cdb, err := gorm.Open(mysql.Open(c.Mysql.PreSource), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("Mysql database init failed: %v", err))
	}
	defer func() {
		cdb1, _ := cdb.DB()
		_ = cdb1.Close()
	}()

	if err := cdb.Exec(fmt.Sprintf("create database if not exists %v CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci",
		c.Mysql.DbName)).Error; err != nil {
		panic(fmt.Sprintf("Mysql database create failed: %v", err))
	}

	db, err := gorm.Open(mysql.Open(c.Mysql.Source), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("Mysql database init failed: %v", err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic(fmt.Sprintf("failed get sql database: %v", err))
	}
	sqlDB.SetMaxIdleConns(int(c.Mysql.MaxIdl))
	sqlDB.SetMaxOpenConns(int(c.Mysql.MaxOpen))
	sqlDB.SetConnMaxLifetime(c.Mysql.ConnMaxLift.AsDuration())

	if err := db.AutoMigrate(
		&orm.GreeterDo{},
		&orm.QuerySceneDo{},
		&orm.QuerySceneParamDo{},
		&orm.QuerySceneWidgetDo{},
	); err != nil {
		panic(fmt.Sprintf("Update Table Failed: %+v", err))
	}

	return db
}

func newRedisDB(c *conf.Data) *redis.ClusterClient {
	addrs := strings.Split(c.Redis.Addr, ",")
	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:         addrs,
		Password:      c.Redis.Password,
		RouteRandomly: true,
		ReadTimeout:   c.GetRedis().ReadTimeout.AsDuration(),
		DialTimeout:   c.GetRedis().ReadTimeout.AsDuration() * 2,
		WriteTimeout:  c.GetRedis().WriteTimeout.AsDuration(),
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		panic(fmt.Sprintf("Redis connection failed: %v", err))
	}

	return rdb
}

func newCKDB(c *conf.Data) *gorm.DB {
	if on, err := strconv.ParseBool(c.GetClickhouse().On); err != nil || !on {
		return nil
	}

	db, err := gorm.Open(clickhouse.Open(c.GetClickhouse().Dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("Clickhouse database init failed: %v", err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic(fmt.Sprintf("failed get sql database: %v", err))
	}
	sqlDB.SetMaxIdleConns(int(c.GetClickhouse().MaxIdl))
	sqlDB.SetMaxOpenConns(int(c.GetClickhouse().MaxOpen))
	sqlDB.SetConnMaxLifetime(c.GetClickhouse().ConnMaxLift.AsDuration())

	return db
}

func newDorisDB(c *conf.Data) *gorm.DB {
	if on, err := strconv.ParseBool(c.GetDoris().GetOn()); err != nil || !on {
		return nil
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.GetDoris().GetUsername(),
		c.GetDoris().GetPassword(),
		c.GetDoris().GetHost(),
		c.GetDoris().GetPort(),
		c.GetDoris().GetDatabase(),
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("Doris database init failed: %v", err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic(fmt.Sprintf("failed get doris sql database: %v", err))
	}
	sqlDB.SetMaxIdleConns(int(c.GetDoris().GetMaxIdl()))
	sqlDB.SetMaxOpenConns(int(c.GetDoris().GetMaxOpen()))
	sqlDB.SetConnMaxLifetime(c.GetDoris().GetConnMaxLift().AsDuration())

	return db
}

func anyDialer(c *conf.Data, logger log.Logger) *grpc.ClientConn {
	conn, err := kgrpc.DialInsecure(
		context.Background(),
		kgrpc.WithEndpoint(c.GetGrpcClient().GetAnyEndpoint()),
		kgrpc.WithMiddleware(
			recovery.Recovery(),
			logging.Client(logger),
		),
		kgrpc.WithTimeout(c.GetGrpcClient().GetTimeout().AsDuration()),
	)
	if err != nil {
		log.Fatalf("discovery micro-fulfillment failed: %v", err)
	}

	return conn
}

// NewData .
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	mysqlDB := newMysqlDB(c)
	redisClient := newRedisDB(c)
	anyConn := anyDialer(c, logger)
	ckDB := newCKDB(c)
	dorisDB := newDorisDB(c)

	cleanup := func() {
		log.Info("closing the data resources")

		_ = anyConn.Close()
		_ = redisClient.Close()

		sqlDB, _ := mysqlDB.DB()
		_ = sqlDB.Close()

		if ckDB != nil {
			sqlDB, _ = ckDB.DB()
			_ = sqlDB.Close()
		}

		if dorisDB != nil {
			sqlDB, _ = dorisDB.DB()
			_ = sqlDB.Close()
		}
	}

	return &Data{
		mysqlDB: mysqlDB,
		ckDB:    ckDB,
		dorisDB: dorisDB,
		rDB:     redisClient,
		anyConn: anyConn,
	}, cleanup, nil
}
