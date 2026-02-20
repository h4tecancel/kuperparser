package bootstrap

import (
	"kuperparser/internal/client"
	"kuperparser/internal/client/proxy"
	"kuperparser/internal/client/transport"
	"kuperparser/internal/config"
	"log/slog"
	"time"
)

func BuildTransport(profile *config.Config, log *slog.Logger, concurrency int) (transport.Transport, error) {
	log.Info("profile",
		"env", profile.Env,
		"proxy_mode", profile.Proxy.Mode,
		"proxy_list_len", len(profile.Proxy.List),
	)

	pvd, failOpen, err := proxy.FromConfig(proxy.Config{
		Mode:               profile.Proxy.Mode,
		List:               profile.Proxy.List,
		RotationURL:        profile.Proxy.RotationURL,
		RotationTTLSeconds: profile.Proxy.RotationTTLSeconds,
		FailOpen:           profile.Proxy.FailOpen,
	}, log)
	if err != nil {
		return nil, err
	}

	proxyFunc := client.ProxyFuncFromProvider(pvd, failOpen, log)

	if proxyFunc == nil {
		log.Warn("proxy OFF", "mode", profile.Proxy.Mode)
	} else {
		log.Info("proxy ON", "mode", profile.Proxy.Mode, "fail_open", profile.Proxy.FailOpen)
	}

	httpClient := client.NewHTTPClientWithProxy(
		time.Duration(profile.HTTP.TimeoutSeconds)*time.Second,
		proxyFunc,
	)

	return transport.Build(transport.Options{
		HTTPClient:  httpClient,
		Retries:     profile.HTTP.Retries,
		Concurrency: concurrency,
		Logger:      log,
	})
}
