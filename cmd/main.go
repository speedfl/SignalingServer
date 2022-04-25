package main

import (
	"github.com/PeerCodeProject/SignalingServer"
	"log"
	"net/http"
	"time"
)

func main() {
	ss := SignalingServer.NewSignalingServer()
	addr := SignalingServer.GetPort()
	server := &http.Server{
		Addr:         addr,
		Handler:      ss,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}
	log.Println("Listening on " + addr)
	err := server.ListenAndServe()

	if err != nil {
		panic(err)
	}
}
