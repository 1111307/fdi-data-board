package data

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/google/wire"
	redis "github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
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
	NewFoDashboardRepo,
	NewDoDashboardRepo,
	NewEtlRepo,
	NewReconcileRepo,
	NewLlmRepo,
	NewSchedulerRedisClient,
)

// Data .
type Data struct {
	conf    *conf.Data
	mysqlDB *gorm.DB
	dorisDB *gorm.DB
	anyConn *grpc.ClientConn

	// 维度值缓存，key: "fo_dashboard:dims", value: *foDimsCache (in dashboard_fo.go)
	dimCache sync.Map
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
		&orm.EtlJobLogDo{},
	); err != nil {
		panic(fmt.Sprintf("Update Table Failed: %+v", err))
	}

	return db
}

func newDorisDB(c *conf.Data) *gorm.DB {
	if c.GetDoris().GetOn() != "true" {
		return nil
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&interpolateParams=true",
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

	if err := sqlDB.Ping(); err != nil {
		panic(fmt.Sprintf("Doris ping failed: %v", err))
	}
	log.Infof("Doris connected: %s:%s", c.GetDoris().GetHost(), c.GetDoris().GetPort())

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

func NewSchedulerRedisClient(c *conf.Data, logger log.Logger) redis.UniversalClient {
	if c.GetRedis() == nil || c.GetRedis().GetAddr() == "" {
		log.NewHelper(logger).Warn("[etl] redis addr is empty, scheduler lock disabled")
		return nil
	}

	addrs := splitAndTrim(c.GetRedis().GetAddr())
	if len(addrs) == 0 {
		log.NewHelper(logger).Warn("[etl] redis addr is empty, scheduler lock disabled")
		return nil
	}

	opts := &redis.UniversalOptions{
		Addrs:        addrs,
		Password:     c.GetRedis().GetPassword(),
		ReadTimeout:  c.GetRedis().GetReadTimeout().AsDuration(),
		WriteTimeout: c.GetRedis().GetWriteTimeout().AsDuration(),
	}
	if strings.EqualFold(c.GetRedis().GetLinkType(), "single") && len(addrs) > 0 {
		opts.Addrs = addrs[:1]
	}

	client := redis.NewUniversalClient(opts)
	pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		log.NewHelper(logger).Errorf("[etl] redis init failed, scheduler lock disabled: %v", err)
		_ = client.Close()
		return nil
	}
	return client
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// NewData .
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	mysqlDB := newMysqlDB(c)
	anyConn := anyDialer(c, logger)
	dorisDB := newDorisDB(c)

	d := &Data{
		conf:    c,
		mysqlDB: mysqlDB,
		dorisDB: dorisDB,
		anyConn: anyConn,
	}

	cleanup := func() {
		log.Info("closing the data resources")

		_ = anyConn.Close()

		sqlDB, _ := mysqlDB.DB()
		_ = sqlDB.Close()

		if dorisDB != nil {
			sqlDB, _ = dorisDB.DB()
			_ = sqlDB.Close()
		}
	}

	return d, cleanup, nil
}
