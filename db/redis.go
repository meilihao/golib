package db

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

func InitRedis(conf *RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:        conf.Addr,
		Password:    conf.Password,
		DB:          conf.DB,
		DialTimeout: 10 * time.Second,
	})

	_, err := client.Ping(context.TODO()).Result()
	if err != nil {
		return nil, err
	}

	return client, nil
}
