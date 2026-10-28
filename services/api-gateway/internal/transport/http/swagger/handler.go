package swagger

import (
	"crypto/sha256"
	"fmt"
	"net/http"

	"github.com/swaggest/swgui/v5emb"
)

type Handler struct {
	specification []byte
	ui            http.Handler
}

func New(config Config, specification []byte) (*Handler, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if len(specification) == 0 {
		return nil, fmt.Errorf("swagger specification is required")
	}
	specificationHash := sha256.Sum256(specification)
	specificationURL := fmt.Sprintf("%s?v=%x", config.SpecificationPath, specificationHash[:8])

	return &Handler{
		specification: specification,
		ui: v5emb.New(
			config.Title,
			specificationURL,
			config.Path,
		),
	}, nil
}

func (h *Handler) OpenAPI(w http.ResponseWriter, _ *http.Request) {
	if h == nil {
		http.Error(w, "swagger handler is not initialized", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	_, _ = w.Write(h.specification)
}

func (h *Handler) UI() http.Handler {
	if h == nil || h.ui == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "swagger handler is not initialized", http.StatusInternalServerError)
		})
	}

	return h.ui
}
