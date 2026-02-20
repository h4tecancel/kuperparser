package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"time"
)

type Transport interface {
	Do(req *http.Request) (*http.Response, error)
}

type Options struct {
	HTTPClient  *http.Client
	Retries     int
	Concurrency int           // ограничение одновременных запросов
	BaseDelay   time.Duration // backoff base
	MaxDelay    time.Duration // backoff max
	Logger      *slog.Logger
}

func (o Options) validate() error {
	if o.HTTPClient == nil {
		return fmt.Errorf("HTTPClient is nil")
	}
	if o.Concurrency < 0 {
		return fmt.Errorf("Concurrency must be >= 0")
	}
	if o.Retries < 0 {
		return fmt.Errorf("Retries must be >= 0")
	}
	return nil
}

func Build(opts Options) (Transport, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}

	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.BaseDelay <= 0 {
		opts.BaseDelay = 300 * time.Millisecond
	}
	if opts.MaxDelay <= 0 {
		opts.MaxDelay = 8 * time.Second
	}

	var t Transport = &HTTPTransport{Client: opts.HTTPClient}

	// retry слой
	if opts.Retries > 0 {
		t = &RetryTransport{
			Base:       t,
			MaxRetries: opts.Retries,
			BaseDelay:  opts.BaseDelay,
			MaxDelay:   opts.MaxDelay,
			Log:        opts.Logger,
		}
	}

	// concurrency слой
	if opts.Concurrency > 0 {
		t = &ConcurrencyTransport{
			Base: t,
			sem:  newSemaphore(opts.Concurrency),
		}
	}

	return t, nil
}

// HTTP transport

type HTTPTransport struct {
	Client *http.Client
}

func (h *HTTPTransport) Do(req *http.Request) (*http.Response, error) {
	return h.Client.Do(req)
}

// semaphore transport

type semaphore struct {
	ch chan struct{}
}

func newSemaphore(n int) *semaphore {
	if n <= 0 {
		n = 1
	}
	return &semaphore{ch: make(chan struct{}, n)}
}

func (s *semaphore) acquire(ctx context.Context) error {
	select {
	case s.ch <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *semaphore) release() {
	<-s.ch
}

type ConcurrencyTransport struct {
	Base Transport
	sem  *semaphore
}

func (t *ConcurrencyTransport) Do(req *http.Request) (*http.Response, error) {
	if err := t.sem.acquire(req.Context()); err != nil {
		return nil, err
	}
	defer t.sem.release()

	return t.Base.Do(req)
}

type RetryTransport struct {
	Base       Transport
	MaxRetries int

	BaseDelay time.Duration
	MaxDelay  time.Duration

	Log *slog.Logger
}

func (r *RetryTransport) Do(req *http.Request) (*http.Response, error) {
	l := r.Log
	if l == nil {
		l = slog.Default()
	}

	var lastErr error

	for attempt := 0; attempt <= r.MaxRetries; attempt++ {
		if err := req.Context().Err(); err != nil {
			return nil, err
		}

		curReq, err := cloneForRetry(req)
		if err != nil {
			return nil, err
		}

		resp, err := r.Base.Do(curReq)
		if err == nil && resp != nil {
			if !shouldRetryStatus(resp.StatusCode) {
				return resp, nil
			}

			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 32*1024))
			_ = resp.Body.Close()

			lastErr = fmt.Errorf("retryable status=%d", resp.StatusCode)

			l.Warn("retryable status",
				"attempt", attempt+1,
				"max_attempts", r.MaxRetries+1,
				"status", resp.StatusCode,
				"url", req.URL.String(),
			)

			if resp.StatusCode == http.StatusTooManyRequests && attempt < r.MaxRetries {
				if d := retryAfterDelay(resp); d > 0 {
					if err := sleepCtx(req.Context(), d); err != nil {
						return nil, err
					}
					continue
				}
			}

		} else {
			if err != nil && !shouldRetryError(err) {
				return nil, err
			}
			lastErr = err

			l.Warn("retryable error",
				"attempt", attempt+1,
				"max_attempts", r.MaxRetries+1,
				"err", err,
				"url", req.URL.String(),
			)
		}

		if attempt == r.MaxRetries {
			break
		}

		d := backoff(r.BaseDelay, r.MaxDelay, attempt)
		if err := sleepCtx(req.Context(), d); err != nil {
			return nil, err
		}
	}

	return nil, lastErr
}

func shouldRetryStatus(code int) bool {
	if code == http.StatusTooManyRequests {
		return true
	}
	return code >= 500 && code <= 599
}

func shouldRetryError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}

func backoff(base, max time.Duration, attempt int) time.Duration {
	if base <= 0 {
		base = 300 * time.Millisecond
	}
	if max <= 0 {
		max = 8 * time.Second
	}

	d := base << attempt
	if d > max {
		d = max
	}

	j := 0.5 + rand.Float64()
	return time.Duration(float64(d) * j)
}

func retryAfterDelay(resp *http.Response) time.Duration {
	ra := resp.Header.Get("Retry-After")
	if ra == "" {
		return 0
	}
	sec, err := strconv.Atoi(ra)
	if err != nil || sec <= 0 {
		return 0
	}
	if sec > 60 {
		sec = 60
	}
	return time.Duration(sec) * time.Second
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func cloneForRetry(req *http.Request) (*http.Request, error) {
	cloned := req.Clone(req.Context())

	if req.Body == nil || req.Body == http.NoBody {
		return cloned, nil
	}
	if req.GetBody == nil {
		return nil, fmt.Errorf("cannot retry request with body: GetBody is nil")
	}
	b, err := req.GetBody()
	if err != nil {
		return nil, fmt.Errorf("cannot retry request with body: GetBody failed: %w", err)
	}
	cloned.Body = b
	return cloned, nil
}
