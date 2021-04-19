// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package gorest

//TODO extract this to https://github.com/mysteriumnetwork/go-rest

var (
	Err500 = NewErrResponse("Internal server error")
)

// Err represents a single error (message) in the ErrResponse.
type Err struct {
	Message string `json:"message"`
}

// ErrResponse represents an error response which may contain one ore more errors.
type ErrResponse struct {
	Errors []Err `json:"errors"`
}

// NewErrResponse creates a new error response containing a single error message.
func NewErrResponse(msg string) *ErrResponse {
	return &ErrResponse{
		Errors: []Err{{Message: msg}},
	}
}
