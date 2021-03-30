// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package db

import "github.com/go-redis/redis/v8"

func New(dbHost, dbPassword string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     dbHost,
		Password: dbPassword,
	})
}
