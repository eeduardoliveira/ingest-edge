// internal/ingest/handler.go
package ingest

import (
	"encoding/json"
	"ingest-edge/internal/mw"
	"ingest-edge/internal/store"
	"net/http"
	"os"
	"strconv"
)

type Handler struct {
	Store       *store.RedisStore
	MaxAccuracy float64
	RL          *mw.RateLimiter
}

func NewHandler(st *store.RedisStore, rl *mw.RateLimiter) *Handler {
	maxAcc := 50.0
	if v := os.Getenv("MAX_ACCURACY_M"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			maxAcc = f
		}
	}
	return &Handler{Store: st, MaxAccuracy: maxAcc, RL: rl}
}

func (h *Handler) Ingest(w http.ResponseWriter, r *http.Request) {
	driverID := r.Context().Value(mw.CtxDriverKey{}).(string)

	var p LocationPoint
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	// Força driver_id do token (não confia no payload)
	p.DriverID = driverID

	if err := p.Validate(h.MaxAccuracy); err != nil {
		http.Error(w, "invalid payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Idempotência (driver+seq)
	ok, err := h.Store.CheckIdempotency(r.Context(), p.DriverID, p.Seq)
	if err != nil {
		http.Error(w, "idem check error", http.StatusInternalServerError)
		return
	}
	if !ok {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"duplicate"}`))
		return
	}

	// Atualiza GEO + heartbeat
	if err := h.Store.UpdateLastPosition(r.Context(), p.DriverID, p.Lat, p.Lng); err != nil {
		http.Error(w, "redis geo error", http.StatusInternalServerError)
		return
	}

	// Fan-out (driver e order)
	if err := h.Store.PublishPoint(r.Context(), p.OrderID, p.DriverID, p); err != nil {
		http.Error(w, "pubsub error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
