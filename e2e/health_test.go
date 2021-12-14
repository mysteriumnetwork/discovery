// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthcheck(t *testing.T) {
	status, err := discoveryAPI.GetStatus()
	assert.NoError(t, err)

	assert.True(t, status.CacheOK)
}
