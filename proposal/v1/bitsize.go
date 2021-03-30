// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package v1

type BitSize float64

const (
	// B is short for Byte.
	B = 1
	// KiB is short for Kibibyte.
	KiB = 1024 * B
	// MiB is short for Mebibyte.
	MiB = 1024 * KiB
	// GiB is short for Gibibyte.
	GiB = 1024 * MiB
	// TiB is short for Tebibyte.
	TiB = 1024 * GiB
	// PiB is short for Pebibyte.
	PiB = 1024 * TiB
	// EiB is short for Exbibyte.
	EiB = 1024 * PiB
)
