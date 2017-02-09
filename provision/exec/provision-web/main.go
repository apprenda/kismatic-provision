package main

import (
	"log"
	"net/http"

	"github.com/sashajeltuhin/ket/provision/openstack"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/nodeup", openstack.NodeUp)
	mux.HandleFunc("/install", openstack.ProvisionAndInstall)
	log.Println("Listening on port 8013")
	err := http.ListenAndServe(":8013", mux)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
