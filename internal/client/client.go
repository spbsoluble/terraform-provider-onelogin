package client

import (
	ol "github.com/onelogin/onelogin-go-sdk/v4/pkg/onelogin"
)

// Client wraps the OneLogin SDK client with provider configuration.
type Client struct {
	SDK    *ol.OneloginSDK
	APIURL string
}
