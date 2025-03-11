package server

import (
	"fmt"
	"net/http"
	"register-power-resources/pkg/apis"
	"strconv"
	"strings"
)

func ListResources(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	resources := registry.GetResources(offset, limit)

	var resourceStrings []string
	for _, resource := range resources {
		resourceStrings = append(resourceStrings, apis.ResourceInfoToString(resource))
	}

	fmt.Fprint(w, strings.Join(resourceStrings, "\n"))
}
