package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sandeep/devinspector/internal/rules"
	"github.com/sandeep/devinspector/internal/scanner"
	"github.com/sandeep/devinspector/internal/utils"
	"github.com/sandeep/devinspector/pkg/models"
)

type scanRequest struct {
	Path string `json:"path"`
}

func Start(port string, logLevel string) error {
	logger := utils.NewLogger(logLevel)
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "DevInspector"})
	})

	mux.HandleFunc("/scan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
			return
		}

		var req scanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		if req.Path == "" {
			req.Path = "."
		}

		cfg, err := utils.LoadConfig(req.Path)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		result, err := scanner.New(rules.EnabledRules(cfg.DisabledRules), cfg.WorkerCount).Scan(req.Path)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, result)
	})

	addr := ":" + port
	logger.Info("DevInspector API listening on %s", addr)
	return http.ListenAndServe(addr, mux)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		fmt.Fprintf(w, `{"error":%q}`, err.Error())
	}
}

var _ models.ScanResult
