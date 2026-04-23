package handler

import (
	"net/http"

	"github.com/sandeep7239/devInspector/pkg/server"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	server.Handler().ServeHTTP(w, r)
}
