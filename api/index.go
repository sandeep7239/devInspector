package handler

import (
	"net/http"

	"github.com/sandeep7239/devInspector/internal/server"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	server.Handler().ServeHTTP(w, r)
}
