package data

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/google/wire"
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
	NewQuerySceneRepo,
	NewDatasourceRepo,
	NewSceneGroupRepo,
	NewFoDashboardRepo,
	NewDoDashboardRepo,
	NewEtlRepo,
)

// Data .
type Data struct {
	mysqlDB *gorm.DB
	dorisDB *gorm.DB
	anyConn *grpc.ClientConn

	// 动态数据源连接缓存，key: datasource_id, value: *gorm.DB
	dsCache sync.Map
	// 维度值缓存，key: "groupID:fieldName", value: dimCacheEntry
	dimCache sync.Map
}

type dimCacheEntry struct {
	values    []string
	expiresAt time.Time
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
		&orm.QueryDatasourceDo{},
		&orm.QuerySceneGroupDo{},
		&orm.QuerySceneDo{},
		&orm.QuerySceneParamDo{},
		&orm.QuerySceneWidgetDo{},
		&orm.SlowQueryLogDo{},
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

	if err := sqlDB.Ping(); err != nil {
		panic(fmt.Sprintf("Doris ping failed: %v", err))
	}
	log.Infof("Doris connected: %s:%s", c.GetDoris().GetHost(), c.GetDoris().GetPort())

	return db
}

// GetDatasourceDB 按需创建并缓存动态数据源连接
func (d *Data) GetDatasourceDB(ctx context.Context, dsID uint64) (*gorm.DB, error) {
	if v, ok := d.dsCache.Load(dsID); ok {
		return v.(*gorm.DB), nil
	}

	var ds orm.QueryDatasourceDo
	if err := d.mysqlDB.WithContext(ctx).
		Where(orm.QueryDatasourceColumns.ID+" = ? AND "+orm.QueryDatasourceColumns.DeleteTime+" IS NULL AND "+orm.QueryDatasourceColumns.Status+" = 1", dsID).
		First(&ds).Error; err != nil {
		return nil, fmt.Errorf("数据源不存在或已禁用: %w", err)
	}

	db, err := buildDatasourceDB(&ds)
	if err != nil {
		return nil, err
	}

	d.dsCache.Store(dsID, db)
	return db, nil
}

// InvalidateDatasourceCache 更新或删除数据源后清除缓存
func (d *Data) InvalidateDatasourceCache(dsID uint64) {
	if v, ok := d.dsCache.LoadAndDelete(dsID); ok {
		if db, ok := v.(*gorm.DB); ok {
			if sqlDB, err := db.DB(); err == nil {
				_ = sqlDB.Close()
			}
		}
	}
}

// buildDatasourceDB 根据数据源配置构建 GORM 连接
func buildDatasourceDB(ds *orm.QueryDatasourceDo) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		ds.Username, ds.Password, ds.Host, ds.Port, ds.DatabaseName,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("数据源连接失败 [%s]: %w", ds.Name, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据源连接池失败: %w", err)
	}
	sqlDB.SetMaxIdleConns(ds.MaxIdl)
	sqlDB.SetMaxOpenConns(ds.MaxOpen)

	return db, nil
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
	anyConn := anyDialer(c, logger)
	dorisDB := newDorisDB(c)

	d := &Data{
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

		// 关闭所有动态缓存的数据源连接
		d.dsCache.Range(func(_, v interface{}) bool {
			if db, ok := v.(*gorm.DB); ok {
				if sqlDB, err := db.DB(); err == nil {
					_ = sqlDB.Close()
				}
			}
			return true
		})
	}

	return d, cleanup, nil
}
