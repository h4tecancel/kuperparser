package jsonfile

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"kuperparser/internal/repository"
)

type Repo struct {
	Path string
	Log  *slog.Logger
}

func New(path string, log *slog.Logger) *Repo {
	if log == nil {
		log = slog.Default()
	}
	return &Repo{Path: path, Log: log}
}

func (r *Repo) Save(ctx context.Context, res repository.CategoryResult) error {
	if err := r.saveAny(ctx, res); err != nil {
		return err
	}
	r.Log.Info("json saved", "path", r.Path, "count", res.Count)
	return nil
}

func (r *Repo) SaveStores(ctx context.Context, res repository.StoresResult) error {
	if err := r.saveAny(ctx, res); err != nil {
		return err
	}
	r.Log.Info("stores json saved", "path", r.Path, "count", res.Count)
	return nil
}

func (r *Repo) saveAny(ctx context.Context, v any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.Path == "" {
		return fmt.Errorf("jsonfile repo: empty path")
	}

	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')

	dir := filepath.Dir(r.Path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	tmp := r.Path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, r.Path); err != nil {
		_ = os.Remove(tmp)
		return err
	}

	return nil
}
