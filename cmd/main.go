package main

import (
	"github.com/PeerCodeProject/SignalingServer/server"
	"github.com/PeerCodeProject/SignalingServer/utils"
	"log"
)

func main() {
	addr := utils.GetPort()
	log.Println("Listening on " + addr)
	err, s := server.RunServer(addr)
	if err != nil {
		s.Close()
		return
	}
}
