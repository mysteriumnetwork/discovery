package db

import "github.com/go-redis/redis/v8"

func New() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
	})
}
