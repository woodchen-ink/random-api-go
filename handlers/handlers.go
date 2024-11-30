package handlers

import (
	"net/http"
	"random-api-go/router"
	"random-api-go/stats"
)

type Router interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

type Handlers struct {
	Stats *stats.StatsManager
}

func (h *Handlers) HandleAPIRequest(w http.ResponseWriter, r *http.Request) {
	HandleAPIRequest(w, r)
}

func (h *Handlers) HandleStats(w http.ResponseWriter, r *http.Request) {
	HandleStats(w, r)
}

func (h *Handlers) HandleURLStats(w http.ResponseWriter, r *http.Request) {
	HandleURLStats(w, r)
}

func (h *Handlers) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	HandleMetrics(w, r)
}

func (h *Handlers) Setup(r *router.Router) {
	r.HandleFunc("/pic/", h.HandleAPIRequest)
	r.HandleFunc("/video/", h.HandleAPIRequest)
	r.HandleFunc("/stats", h.HandleStats)
	r.HandleFunc("/urlstats", h.HandleURLStats)
	r.HandleFunc("/metrics", h.HandleMetrics)
}
