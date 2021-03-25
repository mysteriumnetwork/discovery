package db

import "github.com/go-redis/redis/v8"

func New(dbHost, dbPassword string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     dbHost,
		Password: dbPassword,
	})
}
