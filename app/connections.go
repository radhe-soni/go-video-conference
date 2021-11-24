package app

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v2"
)

const (
	rtcpPLIInterval = 3 * time.Second
)

type Connections struct {
	config webrtc.Configuration
	webrtc *webrtc.API
	// TODO locking
	senderToChannel map[string]chan *webrtc.Track
}

func NewConnections(
	api *webrtc.API,
	config webrtc.Configuration,
) *Connections {
	return &Connections{
		config:          config,
		webrtc:          api,
		senderToChannel: map[string]chan *webrtc.Track{},
	}
}

func (c *Connections) ReceiveTrack(id string, offer *webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	conn, err := c.webrtc.NewPeerConnection(c.config)
	if err != nil {
		return nil, fmt.Errorf("newPeerConnection %w", err)
	}
	if _, ok := c.senderToChannel[id]; !ok {
		c.senderToChannel[id] = make(chan *webrtc.Track, 1)
	}
	conn.AddTrack(<-c.senderToChannel[id])
	conn.SetRemoteDescription(*offer)
	return createAnswer(conn)
}

// user is the caller of the method
// if user connects before peer: since user is first, user will create the channel and track and will pass the track to the channel
// if peer connects before user: since peer came already, he created the channel and is listning and waiting for me to create and pass track
func (c *Connections) CreateTrack(id string, offer *webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	conn, err := c.webrtc.NewPeerConnection(c.config)
	if err != nil {
		return nil, fmt.Errorf("newPeerConnection %w", err)
	}
	if _, err = conn.AddTransceiver(webrtc.RTPCodecTypeVideo); err != nil {
		return nil, fmt.Errorf("adding video transceiver %w", err)
	}
	// Set a handler for when a new remote track starts, this just distributes all our packets
	// to connected peers
	conn.OnTrack(c.newOnTrack(id, conn))

	conn.SetRemoteDescription(*offer)
	return createAnswer(conn)
}

func (c *Connections) newOnTrack(id string, conn *webrtc.PeerConnection) func(*webrtc.Track, *webrtc.RTPReceiver) {
	return func(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
		// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
		// This can be less wasteful by processing incoming RTCP events, then we would emit a NACK/PLI when a viewer requests it
		go func() {
			ticker := time.NewTicker(rtcpPLIInterval)
			for range ticker.C {
				if rtcpSendErr := conn.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: remoteTrack.SSRC()}}); rtcpSendErr != nil {
					log.Println("writeRTCP error", rtcpSendErr)
				}
			}
		}()

		// Create a local track, all our SFU clients will be fed via this track
		// main track of the broadcaster
		localTrack, newTrackErr := conn.NewTrack(remoteTrack.PayloadType(), remoteTrack.SSRC(), "video", "pion")
		if newTrackErr != nil {
			fatalConnError(id, conn, fmt.Errorf("new track error %w", newTrackErr))
			return
		}

		// the channel that will have the local track that is used by the sender
		// the localTrack needs to be fed to the reciever
		localTrackChan := make(chan *webrtc.Track, 1)
		localTrackChan <- localTrack
		if existingChan, ok := c.senderToChannel[id]; ok {
			// feed the exsiting track from user with this track
			existingChan <- localTrack
		} else {
			c.senderToChannel[id] = localTrackChan
		}

		rtpBuf := make([]byte, 1400)
		for { // for publisher only
			i, readErr := remoteTrack.Read(rtpBuf)
			if readErr != nil {
				fatalConnError(id, conn, fmt.Errorf("read error %w", readErr))
				return
			}
			// ErrClosedPipe means we don't have any subscribers, this is ok if no peers have connected yet
			if _, err := localTrack.Write(rtpBuf[:i]); err != nil && err != io.ErrClosedPipe {
				fatalConnError(id, conn, fmt.Errorf("write local error %w", err))
				return
			}
		}
	}
}

func fatalConnError(id string, conn *webrtc.PeerConnection, err error) {
	log.Println("fatalConn error", "id", id, err)
	conn.Close()
}

func createAnswer(conn *webrtc.PeerConnection) (*webrtc.SessionDescription, error) {
	answer, err := conn.CreateAnswer(nil)
	if err != nil {
		return nil, fmt.Errorf("creating answer %w", err)
	}
	if err = conn.SetLocalDescription(answer); err != nil {
		return nil, fmt.Errorf("setting local description %w", err)
	}
	return &answer, nil
}
