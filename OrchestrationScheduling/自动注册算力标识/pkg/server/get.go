package server

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"register-power-resources/pkg/apis"
)

func GetResource(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	registry.RLock()
	resource, ok := registry.Resources[id]
	registry.RUnlock()

	if !ok {
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	}

	fmt.Fprint(w, apis.ResourceInfoToString(resource))
}
