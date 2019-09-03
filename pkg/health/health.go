package health

import (
	"net/http"
	"fmt"
	"github.com/gorilla/mux"
)

// HandleHealth starts health endpoints
func HandleHealth(port string) {
	rtr := mux.NewRouter()
	rtr.HandleFunc(HandleLive()).Methods("GET")
	rtr.HandleFunc(HandleReady()).Methods("GET")
	http.ListenAndServe(port, rtr)
}

// HandleLive returns handler for livness probes
func HandleLive() (string, func(w http.ResponseWriter, req *http.Request)) {
	return "/live", healthResponse()
}

// HandleReady returns handler for readiness proves
func HandleReady() (string, func(w http.ResponseWriter, req *http.Request)) {
	return "/ready", healthResponse()
}

func healthResponse() func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}
}
