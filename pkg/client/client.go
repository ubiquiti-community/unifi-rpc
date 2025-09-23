package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/ubiquiti-community/go-unifi/unifi"
)

type Client interface {
	GetDeviceByMAC(ctx context.Context, site string, mac string) (*unifi.Device, error)
	UpdateDevice(ctx context.Context, site string, d *unifi.Device) (*unifi.Device, error)
	ExecuteCmd(ctx context.Context, site string, mgr string, cmd unifi.Cmd) (any, error)
}

func NewClient(user, pass, apiKey, baseURL string, insecure bool) Client {
	client := unifi.Client{}

	if err := client.SetBaseURL(baseURL); err != nil {
		panic(fmt.Sprintf("failed to set base url: %s", err))
	}

	client.SetAPIKey(apiKey)

	httpClient := &http.Client{}
	httpClient.Transport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,

		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecure,
		},
	}

	jar, _ := cookiejar.New(nil)
	httpClient.Jar = jar

	if err := client.SetHTTPClient(httpClient); err != nil {
		panic(fmt.Sprintf("failed to set http client: %s", err))
	}

	if err := client.Login(context.Background(), user, pass); err != nil {
		panic(fmt.Sprintf("failed to login: %s", err))
	}
	return &client
}
