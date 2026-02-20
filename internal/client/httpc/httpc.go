package httpc

import (
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

func New(timeout time.Duration) *http.Client {
	return NewWithProxy(timeout, nil)
}

func NewWithProxy(timeout time.Duration, proxyFunc func(*http.Request) (*url.URL, error)) *http.Client {
	// тк proxyfunc возвращает ошибку,nil opt => всегда будет nil
	jar, _ := cookiejar.New(nil)

	// TODO: вынести в config
	tr := &http.Transport{
		Proxy: proxyFunc,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,

		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,

		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,

		ForceAttemptHTTP2: true,
	}

	return &http.Client{
		Transport: tr,
		Timeout:   timeout,
		Jar:       jar,
	}
}
