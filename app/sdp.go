package app

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pion/webrtc/v2"
)

// Sdp represent session description protocol describe media communication sessions
type Sdp struct {
	Sdp string
}

func NewSDPHandler(
	connections *Connections,
) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		var (
			err         error
			vars        = mux.Vars(req)
			isSender, _ = strconv.ParseBool(vars["isSender"])
			userID      = vars["userId"]
			peerID      = vars["peerId"]
			session     Sdp
		)
		defer req.Body.Close()
		if err = json.NewDecoder(req.Body).Decode(&session); err != nil {
			jsonError(rw, http.StatusBadRequest, err)
			return
		}

		offer := webrtc.SessionDescription{}
		Decode(session.Sdp, &offer)
		var answer *webrtc.SessionDescription
		if isSender {
			answer, err = connections.CreateTrack(userID, &offer)
		} else {
			answer, err = connections.ReceiveTrack(peerID, &offer)
		}
		if err != nil {
			jsonError(rw, http.StatusInternalServerError, err)
			return
		}
		rw.Header().Add("Content-Type", "application/json")
		json.NewEncoder(rw).Encode(Sdp{Sdp: Encode(*answer)})
	}
}
