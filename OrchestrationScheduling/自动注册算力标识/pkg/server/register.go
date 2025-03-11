package server

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"register-power-resources/pkg/apis"
)

type RequestBody struct {
	ComputeIDs []string `json:"compute_ids"`
}

func RegisterResource(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	var body RequestBody
	err := decoder.Decode(&body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for _, computeID := range body.ComputeIDs {
		resource := apis.ParseResourceInfo(computeID)
		registry.AddResource(resource)
		w.WriteHeader(http.StatusCreated)
		fmt.Println(computeID)
	}
}

func UnregisterResource(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	registry.DeleteResource(id)
	w.WriteHeader(http.StatusNoContent)
	//fmt.Println(registry.Resources)
}
