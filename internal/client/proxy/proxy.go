package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Provider interface {
	Next(ctx context.Context) (string, error)
}

type Mode string

const (
	ModeDisabled Mode = "disabled"
	ModeList     Mode = "list"
	ModeRotation Mode = "rotation"
)

type Config struct {
	Mode               string
	List               []string
	RotationURL        string
	RotationTTLSeconds int
	FailOpen           bool
}

func FromConfig(cfg Config, log *slog.Logger) (Provider, bool, error) {
	if log == nil {
		log = slog.Default()
	}
	mode := strings.ToLower(strings.TrimSpace(cfg.Mode))
	if mode == "" {
		mode = string(ModeDisabled)
	}

	switch Mode(mode) {
	case ModeDisabled:
		return nil, cfg.FailOpen, nil

	case ModeList:
		p, err := NewListProvider(cfg.List)
		if err != nil {
			return nil, cfg.FailOpen, err
		}
		log.Info("proxy enabled", "mode", "list", "count", len(cfg.List), "fail_open", cfg.FailOpen)
		return p, cfg.FailOpen, nil

	case ModeRotation:
		if strings.TrimSpace(cfg.RotationURL) == "" {
			return nil, cfg.FailOpen, fmt.Errorf("proxy.mode=rotation but rotation_url empty")
		}
		ttl := time.Duration(cfg.RotationTTLSeconds) * time.Second
		if ttl <= 0 {
			ttl = 10 * time.Second
		}
		p := NewRotationProvider(cfg.RotationURL, ttl, log)
		log.Info("proxy enabled", "mode", "rotation", "rotation_url", cfg.RotationURL, "ttl", ttl.String(), "fail_open", cfg.FailOpen)
		return p, cfg.FailOpen, nil

	default:
		return nil, cfg.FailOpen, fmt.Errorf("unknown proxy.mode=%q (expected disabled|list|rotation)", cfg.Mode)
	}
}

// fromProvider создаёт proxy func для net/http.Transport.
func FromProvider(p Provider, failOpen bool, log *slog.Logger) func(*http.Request) (*url.URL, error) {
	if p == nil {
		return nil
	}
	if log == nil {
		log = slog.Default()
	}

	return func(req *http.Request) (*url.URL, error) {
		ctx := req.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		raw, err := p.Next(ctx)
		if err != nil {
			log.Warn("proxy provider error", "err", err)
			if failOpen {
				return nil, nil
			}
			return nil, err
		}

		raw = strings.TrimSpace(raw)
		if raw == "" {
			if failOpen {
				return nil, nil
			}
			return nil, fmt.Errorf("empty proxy string")
		}

		if !strings.Contains(raw, "://") {
			raw = "http://" + raw
		}

		u, err := url.Parse(raw)
		if err != nil {
			log.Warn("proxy parse failed", "proxy", raw, "err", err)
			if failOpen {
				return nil, nil
			}
			return nil, err
		}

		log.Debug("proxy selected", "host", u.Host)
		return u, nil
	}
}

type listProvider struct {
	items []string
	idx   uint64
}

func NewListProvider(list []string) (Provider, error) {
	clean := make([]string, 0, len(list))
	for _, s := range list {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		clean = append(clean, s)
	}
	if len(clean) == 0 {
		return nil, fmt.Errorf("proxy list is empty")
	}
	return &listProvider{items: clean}, nil
}

func (p *listProvider) Next(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	i := atomic.AddUint64(&p.idx, 1) - 1
	return p.items[int(i%uint64(len(p.items)))], nil
}

type rotationProvider struct {
	url string
	ttl time.Duration
	log *slog.Logger

	mu      sync.Mutex
	cached  string
	expires time.Time

	client *http.Client
}

func NewRotationProvider(rotationURL string, ttl time.Duration, log *slog.Logger) Provider {
	if log == nil {
		log = slog.Default()
	}
	c := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Proxy: nil,
		},
	}
	return &rotationProvider{
		url:    rotationURL,
		ttl:    ttl,
		log:    log,
		client: c,
	}
}

func (p *rotationProvider) Next(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	now := time.Now()
	p.mu.Lock()
	if p.cached != "" && now.Before(p.expires) {
		out := p.cached
		p.mu.Unlock()
		return out, nil
	}
	p.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("rotation_url status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	proxyStr := parseRotationBody(b)
	proxyStr = strings.TrimSpace(proxyStr)
	if proxyStr == "" {
		return "", fmt.Errorf("rotation_url returned empty proxy")
	}

	p.mu.Lock()
	p.cached = proxyStr
	p.expires = time.Now().Add(p.ttl)
	p.mu.Unlock()

	return proxyStr, nil
}

func parseRotationBody(b []byte) string {
	s := strings.TrimSpace(string(b))
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "{") {
		var m map[string]any
		if json.Unmarshal([]byte(s), &m) == nil {
			for _, k := range []string{"proxy", "url", "data"} {
				if v, ok := m[k]; ok {
					if vs, ok := v.(string); ok {
						return vs
					}
				}
			}
		}
		return ""
	}
	if strings.HasPrefix(s, "[") {
		var arr []any
		if json.Unmarshal([]byte(s), &arr) == nil && len(arr) > 0 {
			if vs, ok := arr[0].(string); ok {
				return vs
			}
		}
		return ""
	}
	return s
}
