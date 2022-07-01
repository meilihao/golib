package db

import (
	"fmt"
	"net/url"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func InitPostgres2Gorm(conf *DBConfig) (*gorm.DB, error) {
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
		postgres.Open(fmt.Sprintf(`host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=%s`,
			conf.Host,
			conf.Port,
			conf.Username,
			conf.Password,
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
