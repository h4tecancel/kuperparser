package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"kuperparser/internal/apis/kuper"
	"kuperparser/internal/apis/kuper/usecases"

	"kuperparser/internal/bootstrap"
	"kuperparser/internal/config"
	httpserver "kuperparser/internal/http-server"

	"kuperparser/internal/logger"
)

func main() {
	var (
		configPath = flag.String("config", "./config/config.yaml", "path to config.yaml")
		host       = flag.String("host", "", "override host")
		port       = flag.Int("port", 0, "override port")
	)
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("load config failed", "err", err)
		os.Exit(1)
	}

	log := logger.New(logger.Options{
		Level:     cfg.Log.Level,
		Format:    cfg.Log.Format,
		AddSource: cfg.Log.AddSource,
		Env:       cfg.Env,
	})
	slog.SetDefault(log)

	if *host != "" {
		cfg.Server.Host = *host
	}
	if *port != 0 {
		cfg.Server.Port = *port
	}

	transport, err := bootstrap.BuildTransport(cfg, log, 10)
	if err != nil {
		log.Error("build transport failed", "err", err)
		os.Exit(1)
	}

	kuperSvc := kuper.New(transport, cfg.Kuper.BaseURL, log)

	usecase := usecases.NewCategoryProductsService(
		kuperSvc,
		cfg.Kuper.BaseURL,
		log,
		cfg.Pagination.PerPage,
		cfg.Pagination.OffersLimit,
		cfg.Pagination.MaxPages,
	)

	api := httpserver.New(log)

	api.RegisterRoutes(httpserver.Deps{
		Categories:     kuperSvc,
		Products:       usecase,
		Store:          kuperSvc,
		DefaultStoreID: cfg.Kuper.StoreID,
		Timeout:        time.Duration(cfg.HTTP.TimeoutSeconds) * time.Second,
	})

	addr := net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port))

	srv := &http.Server{
		Addr:              addr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		log.Info("api started", "addr", addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case sig := <-stop:
		log.Info("shutdown signal received", "signal", sig.String())

		// даём запросам завершиться
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Error("graceful shutdown failed", "err", err)
			_ = srv.Close()
		}
		log.Info("server stopped gracefully")

	case err := <-errCh:

		if errors.Is(err, http.ErrServerClosed) {
			log.Info("server closed")
			return
		}
		log.Error("server stopped with error", "err", err)
		os.Exit(1)
	}
}
