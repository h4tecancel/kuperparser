package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"kuperparser/internal/apis/kuper"
	"kuperparser/internal/apis/kuper/endpoints"
	"kuperparser/internal/bootstrap"
	"kuperparser/internal/config"
	"kuperparser/internal/logger"
	"kuperparser/internal/repository"
	jsonfile "kuperparser/internal/repository/json"
)

// тк апишка требует ни сколько адрес, а стор айди,
// приходится использовать такой костыль на скорую
// руку, мне просто нужны были хотя бы какие нибудь
// валидные айдишники для проверки парсинга, в дальнейшем
// я бы выгружал их более правильным способом, потому что
// подтягиваются далеко не все айдишники.

func main() {
	var (
		configPath = flag.String("config", "./config/config.yaml", "path to config.yaml")
		from       = flag.Int("from", 1, "start storeID (inclusive)")
		to         = flag.Int("to", 20000, "end storeID (inclusive)")
		workers    = flag.Int("workers", 40, "concurrent workers (goroutines)")
		outPath    = flag.String("out", "./output/stores.json", "output json file")
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

	if *from <= 0 || *to <= 0 || *to < *from {
		log.Error("bad range", "from", *from, "to", *to)
		os.Exit(1)
	}
	if *workers <= 0 {
		*workers = 10
	}

	// transport: можно поставить побольше concurrency
	tr, err := bootstrap.BuildTransport(cfg, log, 50)
	if err != nil {
		log.Error("build transport failed", "err", err)
		os.Exit(1)
	}

	kuperSvc := kuper.New(tr, cfg.Kuper.BaseURL, log)

	ids := make(chan int, 1024)
	foundCh := make(chan repository.StoreMeta, 1024)

	var scanned uint64
	var found uint64

	// aggregator
	var storesMu sync.Mutex
	stores := make([]repository.StoreMeta, 0, 4096)

	doneAgg := make(chan struct{})
	go func() {
		defer close(doneAgg)
		for s := range foundCh {
			storesMu.Lock()
			stores = append(stores, s)
			storesMu.Unlock()
		}
	}()

	// workers
	var wg sync.WaitGroup
	ctx := context.Background()

	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range ids {
				atomic.AddUint64(&scanned, 1)

				info, err := kuperSvc.GetStore(ctx, id)
				if err != nil {
					// 404 — просто нет такого storeID
					var he *endpoints.APIError
					if errors.As(err, &he) && he.Status == 404 {
						continue
					}
					// остальное — логируем (но продолжаем)
					log.Warn("GetStore failed", "store_id", id, "err", err)
					continue
				}

				atomic.AddUint64(&found, 1)
				foundCh <- repository.StoreMeta{
					ID:           info.StoreID,
					Name:         info.StoreName,
					Address:      info.StoreAddress,
					RetailerName: info.RetailerName,
				}
			}
		}()
	}

	// progress logger
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			s := atomic.LoadUint64(&scanned)
			f := atomic.LoadUint64(&found)
			log.Info("scan progress", "scanned", s, "found", f, "from", *from, "to", *to)
		}
	}()

	// feed ids
	for id := *from; id <= *to; id++ {
		ids <- id
	}
	close(ids)

	wg.Wait()
	close(foundCh)
	<-doneAgg

	// save
	res := repository.StoresResult{
		FetchedAt: time.Now().UTC().Format(time.RFC3339),
		Stores:    stores,
		Count:     len(stores),
	}
	repo := jsonfile.New(*outPath, log)
	if err := repo.SaveStores(ctx, res); err != nil {
		log.Error("save stores json failed", "err", err)
		os.Exit(1)
	}

	log.Info("done", "scanned", atomic.LoadUint64(&scanned), "found", len(stores), "out", *outPath)
}
