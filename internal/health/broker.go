package health

import (
	"net/http"

	"github.com/gorilla/mux"
)

// BrokerHealth holds logic checking the status of the broker
type BrokerHealth struct {
	port    string
	etcdURL string
}

// NewBrokerProbes creates a new BrokerHealth
func NewBrokerProbes(port string, etcdURL string) *BrokerHealth {
	return &BrokerHealth{
		port:    port,
		etcdURL: etcdURL,
	}
}

// Handle exposes broker's status probes
func (b *BrokerHealth) Handle() {
	rtr := mux.NewRouter()
	rtr.HandleFunc(b.liveProbe()).Methods("GET")
	rtr.HandleFunc(b.readyProbe(b.etcdURL)).Methods("GET")
	http.ListenAndServe(b.port, rtr)
}

func (b *BrokerHealth) liveProbe() (string, func(w http.ResponseWriter, req *http.Request)) {
	return "/live", handleHealth("")
}

func (b *BrokerHealth) readyProbe(etcdURL string) (string, func(w http.ResponseWriter, req *http.Request)) {
	return "/ready", handleHealth(etcdURL)
}
