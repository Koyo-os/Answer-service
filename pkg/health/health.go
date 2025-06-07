package health

import (
	"context"
	"net/http"

	"github.com/Koyo-os/answer-service/pkg/logger"
	"go.uber.org/zap"
)

type (
	Healther interface {
		IsHealthy() bool
	}

	HealthCheker struct {
		healthers []Healther
		logger    *logger.Logger
		server *http.Server
	}
)

func NewHealthChecker(healthers ...Healther) *HealthCheker {
	return &HealthCheker{
		logger:    logger.Get(),
		healthers: healthers,
		server: &http.Server{},
	}
}

func (h *HealthCheker) Close(ctx context.Context) error {
	return h.server.Shutdown(ctx)
}

func (h *HealthCheker) HeathHandler(w http.ResponseWriter, r *http.Request) {
	healthy := true

	for _, healther := range h.healthers {
		if !healther.IsHealthy() {
			healthy = false
		}
	}

	if healthy {
		w.Write([]byte("OK"))
	} else {
		w.Write([]byte("UNHEALTHY"))
	}
}

func (h *HealthCheker) RunServer(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.HeathHandler)
	
	h.server.Addr = addr
	h.server.Handler = mux

	if err := h.server.ListenAndServe(); err != nil {
		h.logger.Error("error run health server",
			zap.String("addr", addr),
			zap.Error(err))

		return
	}

}
