package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"kuperparser/internal/http-server/respond"
	"runtime/debug"

	"log/slog"
	"net/http"
	"time"
)

type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytes += n
	return n, err
}

func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const hdr = "X-Request-Id"
		rid := r.Header.Get(hdr)
		if rid == "" {
			rid = NewRID()
		}

		r.Header.Set(hdr, rid)
		w.Header().Set(hdr, rid)

		next.ServeHTTP(w, r)
	})
}

func AccessLog(log *slog.Logger, next http.Handler) http.Handler {
	if log == nil {
		log = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w}

		next.ServeHTTP(sw, r)

		if sw.status == 0 {
			sw.status = http.StatusOK
		}

		d := time.Since(start)
		log.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"status", sw.status,
			"bytes", sw.bytes,
			"duration_ms", d.Milliseconds(),
			"remote", r.RemoteAddr,
			"rid", r.Header.Get("X-Request-Id"),
		)
	})
}

func RecoverPanic(log *slog.Logger, next http.Handler) http.Handler {
	if log == nil {
		log = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				log.Error("panic recovered",
					"panic", v,
					"rid", r.Header.Get("X-Request-Id"),
					"stack", string(debug.Stack()),
				)
				respond.WriteError(w, http.StatusInternalServerError, "internal_error", "panic")

			}
		}()
		next.ServeHTTP(w, r)
	})
}

func NewRID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
