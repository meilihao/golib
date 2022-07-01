package db

import (
	"fmt"
	"net/url"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"xorm.io/xorm"
)

func InitMySQL2Xorm(conf *DBConfig) (*xorm.Engine, error) {
	if conf.Loc == "" {
		conf.Loc = url.QueryEscape("Asia/Shanghai")
	}

	engine, err := xorm.NewEngine("mysql",
		fmt.Sprintf(`%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=%s`,
			conf.Username,
			conf.Password,
			conf.Host,
			conf.Port,
			conf.Name,
			conf.Loc))
	if err != nil {
		return nil, err
	}

	if conf.Loc == "UTC" {
		engine.DatabaseTZ = time.UTC
		engine.TZLocation = time.UTC
	}

	engine.SetMaxOpenConns(conf.MaxOpenConns)
	engine.SetMaxIdleConns(conf.MaxIdleConns)
	engine.SetConnMaxLifetime(time.Hour * 7)
	engine.ShowSQL(conf.ShowSQL)

	if err = engine.Ping(); err != nil {
		return nil, err
	}

	return engine, nil
}

func InitMySQL2Gorm(conf *DBConfig) (*gorm.DB, error) {
	if conf.Loc == "" {
		conf.Loc = url.QueryEscape("Asia/Shanghai")
	}

	gconf := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	}
	if conf.ShowSQL {
		gconf.Logger = logger.Default.LogMode(logger.Info)
	}

	engine, err := gorm.Open(
		mysql.Open(fmt.Sprintf(`%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=%s`,
			conf.Username,
			conf.Password,
			conf.Host,
			conf.Port,
			conf.Name,
			conf.Loc)), gconf)
	if err != nil {
		return nil, err
	}

	sqlDB, _ := engine.DB()

	sqlDB.SetMaxOpenConns(conf.MaxOpenConns)
	sqlDB.SetMaxIdleConns(conf.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Hour * 7)

	if err = sqlDB.Ping(); err != nil {
		return nil, err
	}

	return engine, nil
}
