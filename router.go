package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
)

func newRouter() *mux.Router {
	var router = mux.NewRouter()
	router.HandleFunc("/status", statusHandler).Methods("GET")
	return router
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"app":     app.Name,
		"version": app.Version,
		"status":  "OK",
	}

	if err := daemonContext.findPIDFile(); err == nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}
