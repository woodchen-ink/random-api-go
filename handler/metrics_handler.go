package handler

import (
	"encoding/json"
	"net/http"
	"random-api-go/monitoring"
)

func HandleMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := monitoring.CollectMetrics()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}
