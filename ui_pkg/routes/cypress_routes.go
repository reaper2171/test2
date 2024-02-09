package routes

import (
	"log"
	"net/http"

	"bitbucket.com/testing-cypress-server/server1/pkg/controller"

	"github.com/gorilla/mux"
)

func InitRoutes() {
	r := mux.NewRouter()

	r.HandleFunc("/run-cypress", controller.StartTest)

	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(":8081", nil))
}
