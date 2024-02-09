package routes

import (
	"hub/pkg/controller"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func RegisterRoutes() {
	r := mux.NewRouter()

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Handle the root endpoint, serving an HTML page
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	}).Methods("GET")

	r.HandleFunc("/startTest", controller.StartCypressTest).Methods("POST")
	r.HandleFunc("/cancelRequest", controller.CancelRequest).Methods("POST")

	//For pod redis
	r.HandleFunc("/updateRedisFromPod", controller.UpdateRedisFromPod).Methods("POST")

	//For rerun
	r.HandleFunc("/rerunTest", controller.StartCypressRerunTest).Methods("POST")

	r.HandleFunc("/getReports/{startDate}/{endDate}", controller.GetReports).Methods("GET")

	r.HandleFunc("/jobId/{job_id}", controller.GetJobReport).Methods("GET")
	r.HandleFunc("/getTodayReports", controller.GetJobReportToday).Methods("GET")

	http.Handle("/", r)
	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
