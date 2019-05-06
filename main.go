package main

import (
	"math/rand"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/adaickalavan/Go-WebRTC-GStreamer/handler"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v2"
)

var peerConnectionConfig = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			URLs: []string{"stun:stun.l.google.com:19302"},
		},
	},
}

const (
	rtcpPLIInterval = time.Second * 3
)

func init() {
	//Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	// Everything below is the pion-WebRTC API, thanks for using it ❤️.
	// Create a MediaEngine object to configure the supported codec
	m := webrtc.MediaEngine{}

	// Setup the codecs you want to use.
	// Only support VP8, this makes our proxying code simpler
	m.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))

	// Create the API object with the MediaEngine
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))

	// Run SDP server
	s := newSDPServer(api)
	s.run(os.Getenv("LISTENINGADDR"))
}

type pcinfo struct {
	pc    *webrtc.PeerConnection
	track *webrtc.Track
	ssrc uint32
}

type sdpServer struct {
	recoverCount int
	api          *webrtc.API
	pcUpload     map[string]*pcinfo
	pcDownload   map[string]*webrtc.PeerConnection
	mux          *http.ServeMux
}

func newSDPServer(api *webrtc.API) *sdpServer {
	return &sdpServer{
		api:        api,
		pcUpload:   make(map[string]*pcinfo),
		pcDownload: make(map[string]*webrtc.PeerConnection),
	}
}

func (s *sdpServer) makeMux() {
	mux := http.NewServeMux()
	mux.HandleFunc("/sdp", handlerSDP(s))
	mux.HandleFunc("/join", handlerJoin)
	mux.HandleFunc("/publish", handlerPublish)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	s.mux = mux
}

func (s *sdpServer) run(port string) {
	defer func() {
		s.recoverCount++
		if s.recoverCount > 1 {
			log.Fatal("signal.runSDPServer(): Failed to run")
		}
		if r := recover(); r != nil {
			log.Println("signal.runSDPServer(): PANICKED AND RECOVERED")
			log.Println("Panic:", r)
			go s.run(port)
		}
	}()

	s.makeMux()

	server := &http.Server{
		Addr:           ":" + port,
		Handler:        s.mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

type message struct {
	Name string                    `json:"name"`
	SD   webrtc.SessionDescription `json:"sd"`
}

func handlerSDP(s *sdpServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var offer message
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&offer); err != nil {
			handler.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		// Create a new RTCPeerConnection
		pc, err := s.api.NewPeerConnection(peerConnectionConfig)
		if err != nil {
			panic(err)
		}

		switch strings.Split(offer.Name, ":")[0] {
		case "Publisher":
			// Allow us to receive 1 video track
			if _, err = pc.AddTransceiver(webrtc.RTPCodecTypeVideo); err != nil {
				panic(err)
			}

			// Store the pc handle
			var localTrack = &webrtc.Track{}
			var err error
			var ssrcLocal uint32
			if k, ok := s.pcUpload[offer.Name]; !ok {
				ssrcLocal = rand.Uint32()   
				localTrack, err = pc.NewTrack(webrtc.DefaultPayloadTypeVP8, ssrcLocal, "video", "pion")
				if err != nil {
					log.Panic(err)
				}
				s.pcUpload[offer.Name] = &pcinfo{pc: pc, track: localTrack, ssrc: ssrcLocal}
				log.Println("new publisher")
			} else {
				localTrack = k.track
				ssrcLocal = k.ssrc
				log.Println("old publisher")
			}
			// Set a handler for when a new remote track starts
			// Add the incoming track to the list of tracks maintained in the server
			addOnTrack(pc, localTrack, ssrcLocal)

			log.Println("Publisher")
			log.Println(s.pcUpload)
			log.Println(s.pcDownload)

		case "Client":
			if len(s.pcUpload) == 0 {
				handler.RespondWithError(w, http.StatusInternalServerError, "No local track available for peer connection")
				return
			}

			// Store the pc handle
			s.pcDownload[offer.Name] = pc

			for _, v := range s.pcUpload {
				_, err = pc.AddTrack(v.track)
				if err != nil {
					handler.RespondWithError(w, http.StatusInternalServerError, "Unable to add local track to peer connection")
					return
				}
				break
			}

			log.Println("Client")
			log.Println(s.pcUpload)
			log.Println(s.pcDownload)

		default:
			handler.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		// Set the remote SessionDescription
		err = pc.SetRemoteDescription(offer.SD)
		if err != nil {
			panic(err)
		}

		// Create answer
		answer, err := pc.CreateAnswer(nil)
		if err != nil {
			panic(err)
		}

		// Sets the LocalDescription, and starts our UDP listeners
		err = pc.SetLocalDescription(answer)
		if err != nil {
			panic(err)
		}

		handler.RespondWithJSON(w, http.StatusAccepted,
			map[string]interface{}{
				"Result": "Successfully received incoming client SDP",
				"SD":     answer,
			})
	}
}

func addOnTrack(pc *webrtc.PeerConnection, localTrack *webrtc.Track, ssrcRemote uint32) {
	// Set a handler for when a new remote track starts, this just distributes all our packets
	// to connected peers
	pc.OnTrack(func(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
		// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
		// This can be less wasteful by processing incoming RTCP events, then we would emit a NACK/PLI when a viewer requests it
		go func() {
			ticker := time.NewTicker(rtcpPLIInterval)
			for range ticker.C {
				if rtcpSendErr := pc.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: remoteTrack.SSRC()}}); rtcpSendErr != nil {
					log.Println(rtcpSendErr)
				}
			}
		}()

		// Create a local track, all our SFU clients will be fed via this track
		// var newTrackErr error
		// localTrackNew, newTrackErr := pc.NewTrack(remoteTrack.PayloadType(), ssrcRemote, "video", "pion")
		// if newTrackErr != nil {
		// 	panic(newTrackErr)
		// }
		// log.Println(remoteTrack.PayloadType())
		// log.Println(webrtc.DefaultPayloadTypeVP8)
		// *localTrack = *localTrackNew

		// rtpBuf := make([]byte, 1400)
		// for {
		// 	i, readErr := remoteTrack.Read(rtpBuf)
		// 	if readErr != nil {
		// 		panic(readErr)
		// 	}

		// 	// ErrClosedPipe means we don't have any subscribers, this is ok if no peers have connected yet
		// 	if _, err := localTrack.Write(rtpBuf[:i]); err != nil && err != io.ErrClosedPipe {
		// 		panic(err)
		// 	}
		// }
		
		log.Println("Track acquired", remoteTrack.Kind(), remoteTrack.Codec())
		for {
			rtp, err := remoteTrack.ReadRTP()
			if err != nil {
				log.Panic(err)
			}
			rtp.SSRC = ssrcRemote
			rtp.PayloadType = webrtc.DefaultPayloadTypeVP8

			if err := localTrack.WriteRTP(rtp); err != nil && err != io.ErrClosedPipe {
				log.Panic(err)
			}
		}
	})
}

func handlerJoin(w http.ResponseWriter, r *http.Request) {
	handler.Push(w, "./static/js/join.js")
	tpl, err := template.ParseFiles("./template/join.html")
	if err != nil {
		log.Printf("\nParse error: %v\n", err)
		handler.RespondWithError(w, http.StatusInternalServerError, "ERROR: Template parse error.")
		return
	}
	handler.Render(w, r, tpl, nil)
}

func handlerPublish(w http.ResponseWriter, r *http.Request) {
	handler.Push(w, "./static/js/publish.js")
	tpl, err := template.ParseFiles("./template/publish.html")
	if err != nil {
		log.Printf("\nParse error: %v\n", err)
		handler.RespondWithError(w, http.StatusInternalServerError, "ERROR: Template parse error.")
		return
	}
	handler.Render(w, r, tpl, nil)
}
