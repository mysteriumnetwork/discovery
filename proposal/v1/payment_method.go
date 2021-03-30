// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package v1

import (
	"math/big"
	"time"
)

// PaymentMethod is a interface for all types of payment methods
type PaymentMethod struct {
	Price    Money         `json:"price"`
	Duration time.Duration `json:"duration"`
	Bytes    int           `json:"bytes"`
	Type     string        `json:"type"`
}

func (pm PaymentMethod) pricePerHour() *big.Int {
	pricePerHour := new(big.Int)
	perDuration := new(big.Float).SetInt64(int64(pm.Duration))
	val, _ := new(big.Float).Mul(
		new(big.Float).SetInt(pm.Price.Amount),
		new(big.Float).Quo(
			new(big.Float).SetInt64(int64(time.Hour)),
			perDuration,
		),
	).Int(pricePerHour)
	return val
}

func (pm PaymentMethod) pricePerGiB() *big.Int {
	pricePerGib := new(big.Int)
	perBytes := new(big.Float).SetInt64(int64(pm.Bytes))
	val, _ := new(big.Float).Mul(
		new(big.Float).SetInt(pm.Price.Amount),
		new(big.Float).Quo(
			big.NewFloat(GiB),
			perBytes,
		),
	).Int(pricePerGib)
	return val
}
