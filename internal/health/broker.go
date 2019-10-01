package health

import (
	"net/http"
)

// BrokerLiveProbe returns handler for livness probes for broker container
func BrokerLiveProbe() (string, func(w http.ResponseWriter, req *http.Request)) {
	return "/live", handleHealth("")
}

// BrokerReadyProbe returns handler for readiness proves
func BrokerReadyProbe(etcdURL string) (string, func(w http.ResponseWriter, req *http.Request)) {
	return "/ready", handleHealth(etcdURL)
}
