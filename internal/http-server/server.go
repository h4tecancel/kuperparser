package httpserver

import (
	"kuperparser/internal/http-server/handlers/categories"
	"kuperparser/internal/http-server/handlers/products"
	"kuperparser/internal/http-server/middleware"
	"log/slog"
	"net/http"
	"time"
)

type Server struct {
	log *slog.Logger
	mux *http.ServeMux
}

func New(log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}
	return &Server{log: log, mux: http.NewServeMux()}
}

func (s *Server) Handler() http.Handler {
	var h http.Handler = s.mux
	h = middleware.WithRequestID(h)
	h = middleware.RecoverPanic(s.log, h)
	h = middleware.AccessLog(s.log, h)
	return h
}

type Deps struct {
	Categories     categories.Lister
	Products       products.ProductsGetter
	Store          products.StoreGetter
	DefaultStoreID int
	Timeout        time.Duration
}

func (s *Server) RegisterRoutes(dep Deps) {

	s.mux.HandleFunc("/categories", categories.NewGetHandler(categories.Options{
		Log:            s.log,
		Lister:         dep.Categories,
		DefaultStoreID: dep.DefaultStoreID,
		Timeout:        dep.Timeout,
		HideRoofLeaf:   true,
	}))

	s.mux.HandleFunc("/products", products.NewGetHandler(products.Options{
		Log:            s.log,
		Products:       dep.Products,
		Store:          dep.Store,
		DefaultStoreID: dep.DefaultStoreID,
		Timeout:        dep.Timeout,
	}))
}
