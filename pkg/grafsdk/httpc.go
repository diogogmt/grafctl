package grafsdk

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

var (
	defaultNonVerifyingTransport http.RoundTripper = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			DualStack: true,
		}).DialContext,
		TLSHandshakeTimeout:   15 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   5,
		IdleConnTimeout:       5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}
	defaultClient = &http.Client{
		Timeout:   120 * time.Second,
		Transport: defaultNonVerifyingTransport,
	}
)

type HTTPClient struct {
	client  *http.Client
	headers map[string]string
}

func NewHTTPClient(ctx context.Context) *HTTPClient {
	return &HTTPClient{
		client:  defaultClient,
		headers: make(map[string]string),
	}
}

func (c *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *HTTPClient) SetHeaders(headers map[string]string) {
	for k, v := range headers {
		c.headers[k] = v
	}
}
