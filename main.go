package main

import (
	"log"
	"os"

	"github.com/pion/webrtc/v2"
	"github.com/soldiermoth/go-video-conference/app"
)

func main() {
	log.SetOutput(os.Stderr)

	m := webrtc.MediaEngine{}
	// Setup the codecs you want to use.
	// Only support VP8(video compression), this makes our proxying code simpler
	m.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))
	connections := app.NewConnections(
		webrtc.NewAPI(webrtc.WithMediaEngine(m)),
		webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{{
				// TODO: Why this one?
				URLs: []string{"stun:stun.l.google.com:19302"},
			}},
		})
	server := app.NewHTTP(connections)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
