package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"time"

	"kuperparser/internal/apis/kuper"
	"kuperparser/internal/apis/kuper/usecases"

	"kuperparser/internal/bootstrap"
	"kuperparser/internal/config"
	"kuperparser/internal/logger"
	"kuperparser/internal/repository"
	jsonfile "kuperparser/internal/repository/json"
)

func main() {
	var (
		configPath = flag.String("config", "./config/config.yaml", "path to config.yaml")
		storeID    = flag.Int("storeID", 86, "override storeID (optional)")
		categoryID = flag.Int("categoryID", 0, "override categoryID (optional)")
		outputFile = flag.String("out", "", "override output file (optional)")
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
	})
	slog.SetDefault(log)

	// overrides
	if *storeID > 0 {
		cfg.Kuper.StoreID = *storeID
	}
	if *categoryID > 0 {
		cfg.CLI.CategoryID = *categoryID
	}
	if *outputFile != "" {
		cfg.CLI.OutputFile = *outputFile
	}

	if cfg.Kuper.StoreID <= 0 {
		log.Error("store_id must be > 0 (set in config.yaml or via -storeID)")
		os.Exit(1)
	}
	if cfg.CLI.CategoryID <= 0 {
		log.Error("category_id must be > 0 (set in config.yaml or via -categoryID)")
		os.Exit(1)
	}
	if cfg.CLI.OutputFile == "" {
		log.Error("output_file must not be empty (set in config.yaml or via -out)")
		os.Exit(1)
	}

	// единая сборка транспорта
	transport, err := bootstrap.BuildTransport(cfg, log, 5)
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

	repo := jsonfile.New(cfg.CLI.OutputFile, log)

	// общий timeout на задачу парсинга
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.HTTP.TimeoutSeconds)*time.Second)
	defer cancel()

	// мета магазина
	var storeMeta *repository.StoreMeta
	if info, err := kuperSvc.GetStore(ctx, cfg.Kuper.StoreID); err != nil {
		log.Warn("get store info failed (continue)", "err", err, "store_id", cfg.Kuper.StoreID)
	} else {
		storeMeta = &repository.StoreMeta{
			ID:           info.StoreID,
			Name:         info.StoreName,
			Address:      info.StoreAddress,
			RetailerName: info.RetailerName,
		}
	}

	products, slug, err := usecase.GetByCategoryID(ctx, cfg.Kuper.StoreID, cfg.CLI.CategoryID)
	if err != nil {
		log.Error("parse category failed", "err", err)
		os.Exit(1)
	}

	res := repository.CategoryResult{
		FetchedAt: time.Now().UTC().Format(time.RFC3339),
		Store:     storeMeta,
		Category: &repository.CategoryMeta{
			ID:   cfg.CLI.CategoryID,
			Slug: slug,
		},
		Products: products,
		Count:    len(products),
	}

	if err := repo.Save(ctx, res); err != nil {
		log.Error("save json failed", "err", err)
		os.Exit(1)
	}

	log.Info("done",
		"env", cfg.Env,
		"store_id", cfg.Kuper.StoreID,
		"category_id", cfg.CLI.CategoryID,
		"slug", slug,
		"count", len(products),
		"output", cfg.CLI.OutputFile,
	)
}
