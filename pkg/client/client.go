package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"slices"
	"sync"
	"time"

	"github.com/ubiquiti-community/go-unifi/unifi"
)

type Client interface {
	GetDeviceByMAC(ctx context.Context, mac string) (*unifi.Device, error)
	UpdateDevice(ctx context.Context, d *unifi.Device) (*unifi.Device, error)
	PowerCycle(ctx context.Context, mac string, port int) error
}

func NewClient(baseURL, user, pass string, insecure bool) Client {
	return &lazyClient{
		baseURL:  baseURL,
		user:     user,
		pass:     pass,
		insecure: insecure,
		jar:      nil,
	}
}

type lazyClient struct {
	baseURL  string
	user     string
	pass     string
	insecure bool
	jar      *cookiejar.Jar

	once  sync.Once
	inner *unifi.Client
}

func (c *lazyClient) GetDeviceByMAC(ctx context.Context, mac string) (*unifi.Device, error) {
	if err := c.init(ctx); err != nil {
		return nil, err
	}
	return c.inner.GetDeviceByMAC(ctx, "default", mac)
}

func (c *lazyClient) UpdateDevice(ctx context.Context, d *unifi.Device) (*unifi.Device, error) {
	if err := c.init(ctx); err != nil {
		return nil, err
	}
	return c.inner.UpdateDevice(ctx, "default", d)
}

func (c *lazyClient) PowerCycle(ctx context.Context, mac string, port int) error {
	if err := c.init(ctx); err != nil {
		return err
	}
	if _, err := c.inner.ExecuteCmd(ctx, "default", "devmgr", unifi.Cmd{
		Command: "power-cycle",
		MAC:     mac,
		PortIDX: &port,
	}); err != nil {
		log.Printf("[ERROR] failed to power cycle: %s", err)
		return err
	}
	return nil
}

func setHTTPClient(c *unifi.Client, insecure bool, jar *cookiejar.Jar) {
	httpClient := &http.Client{}
	httpClient.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
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

	httpClient.Jar = jar

	err := c.SetHTTPClient(httpClient)
	if err != nil {
		panic(fmt.Sprintf("failed to set http client: %s", err))
	}
}

var initErr error

func (c *lazyClient) init(ctx context.Context) error {
	if c.jar == nil {
		c.jar, _ = cookiejar.New(nil)
	}

	c.once.Do(func() {
		c.inner = &unifi.Client{}
		setHTTPClient(c.inner, c.insecure, c.jar)

		initErr = c.inner.SetBaseURL(c.baseURL)
		if initErr != nil {
			return
		}

		initErr = c.inner.Login(ctx, c.user, c.pass)
		if initErr != nil {
			return
		}
		log.Printf("[TRACE] Unifi controller version: %q", c.inner.Version())
	})

	if c.isTokenExpired() {
		log.Printf("[TRACE] Unifi controller token expired, reinitializing")
		initErr = c.inner.Login(ctx, c.user, c.pass)
	}

	return initErr
}

func (c *lazyClient) isTokenExpired() bool {
	if c.jar == nil {
		c.jar, _ = cookiejar.New(nil)
	}

	unifiURL, err := url.Parse(c.baseURL)
	if err != nil {
		return true
	}

	cookies := c.jar.Cookies(unifiURL)

	if len(cookies) == 0 {
		return true
	}

	return slices.IndexFunc(cookies, func(c *http.Cookie) bool {
		return c.Name == "TOKEN"
	}) == -1
}

func (c *lazyClient) Version() string {
	if err := c.init(context.Background()); err != nil {
		panic(fmt.Sprintf("client not initialized: %s", err))
	}
	return c.inner.Version()
}
