// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package location

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type locationProvider struct {
	address string
	user    string
	pass    string
}

// NewLocationProvider create a new location provider instance.
func NewLocationProvider(address, user, pass string) *locationProvider {
	return &locationProvider{
		address: address,
		user:    user,
		pass:    pass,
	}
}

// Country returns a country code for the provided IP-address.
func (lp *locationProvider) Country(ip string) (countryCode string, err error) {
	url := fmt.Sprintf("%s/%s", lp.address, ip)

	req, err := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(lp.user, lp.pass)

	ctx, cancel := context.WithTimeout(req.Context(), 5*time.Second)
	defer cancel()

	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	var country struct {
		Country string `json:"country"`
	}

	err = json.NewDecoder(resp.Body).Decode(&country)
	if err != nil {
		return "", err
	}

	return country.Country, nil
}
