// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package db

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
)

func New(dbConnString string) (*pgxpool.Pool, error) {
	return pgxpool.Connect(context.Background(), dbConnString)
}
