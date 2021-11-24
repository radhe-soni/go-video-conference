package app

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

func NewHTTP(connections *Connections) *http.Server {
	r := mux.NewRouter()

	// SDP Handler
	r.NewRoute().
		Methods(http.MethodPost).
		Path("/webrtc/sdp/m/{meetingID}/c/{userID}/p/{peerID}/s/{isSender}").
		Handler(NewSDPHandler(connections))

	r.NewRoute().
		Methods(http.MethodGet, http.MethodOptions, http.MethodHead).
		Handler(http.FileServer(http.Dir("./ui")))

	return &http.Server{
		Handler:      r,
		Addr:         ":8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
}

func jsonError(rw http.ResponseWriter, status int, err error) {
	log.Println("writing error", "status", status, "error", err)
	rw.Header().Add("Content-Type", "application/json")
	rw.WriteHeader(status)
	json.NewEncoder(rw).Encode(struct {
		Error string `json:"error"`
	}{
		Error: err.Error(),
	})
}
