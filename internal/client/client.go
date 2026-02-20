package client

import (
	"kuperparser/internal/client/httpc"
	"kuperparser/internal/client/proxy"
	"kuperparser/internal/client/transport"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

type Transport = transport.Transport

type Options struct {
	HTTPClient *http.Client
	Retries    int
	Workers    int

	BaseDelay time.Duration
	MaxDelay  time.Duration

	Logger *slog.Logger
}

func Build(opts Options) (Transport, error) {
	return transport.Build(transport.Options{
		HTTPClient:  opts.HTTPClient,
		Retries:     opts.Retries,
		Concurrency: opts.Workers,
		BaseDelay:   opts.BaseDelay,
		MaxDelay:    opts.MaxDelay,
		Logger:      opts.Logger,
	})
}

func NewHTTPClient(timeout time.Duration) *http.Client {
	return httpc.New(timeout)
}

func NewHTTPClientWithProxy(timeout time.Duration, proxyFunc func(*http.Request) (*url.URL, error)) *http.Client {
	return httpc.NewWithProxy(timeout, proxyFunc)
}

func ProxyFuncFromProvider(p proxy.Provider, failOpen bool, log *slog.Logger) func(*http.Request) (*url.URL, error) {
	return proxy.FromProvider(p, failOpen, log)
}
