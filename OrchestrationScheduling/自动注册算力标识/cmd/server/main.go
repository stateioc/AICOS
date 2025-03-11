package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"register-power-resources/pkg/server"
)

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/resources", server.RegisterResource).Methods("POST")
	router.HandleFunc("/resources/{id}", server.UnregisterResource).Methods("DELETE")
	router.HandleFunc("/resources/{id}", server.GetResource).Methods("GET")
	router.HandleFunc("/resources", server.ListResources).Methods("GET")

	fmt.Println("Starting server at :8080")
	http.ListenAndServe(":8080", router)
	log.Fatal(http.ListenAndServe(":8080", router))
}
