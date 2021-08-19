package db

import (
	"fmt"
	"net/url"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"xorm.io/xorm"
)

type DBConfig struct {
	Host         string
	Port         int
	Name         string
	Username     string
	Password     string
	MaxIdleConns int    `yaml:"max_idle_conns"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	ShowSQL      bool   `yaml:"show_sql"`
	Loc          string `yaml:"loc"`
}

func InitMySQL2Xorm(conf *DBConfig) (*xorm.Engine, error) {
	if conf.Loc == "" {
		conf.Loc = url.QueryEscape("Asia/Shanghai")
	}

	engine, err := xorm.NewEngine("mysql",
		fmt.Sprintf(`%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=%s`,
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
