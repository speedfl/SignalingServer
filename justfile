set shell := ["cmd.exe", "/c"]

build:
    go build -o serv github.com/PeerCodeProject/SignalingServer

exec: build
    serv

run:
    go run cmd/main.go

sloc:
  wc -l **/*.go