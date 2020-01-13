package health

import (
	"fmt"
	"net/http"
)

func handleHealth(etcdURL string) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		if etcdURL != "" {
			resp, err := http.Get(fmt.Sprintf("%s%s", etcdURL, "/health"))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
		return
	}
}
