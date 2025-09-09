package adapters

import (
	"crypto/tls"

	"github.com/go-resty/resty/v2"
)

type RestyClientAdapter struct {
	Client *resty.Client
}

// NewRestyClientAdapter creates a new resty client adapter
func NewRestyClientAdapter() *RestyClientAdapter {
	client := resty.New()
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	return &RestyClientAdapter{
		Client: client,
	}
}

// NewRestyClientAdapterRequest creates a new resty client adapter request
func NewRestyClientAdapterRequest() *resty.Request {
	return resty.New().R()
}

// R retruns the resty request
func (r *RestyClientAdapter) R() *resty.Request {
	return r.Client.R()
}
