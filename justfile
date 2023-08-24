#set shell := ["cmd.exe", "/c"]

build:
    go build -o serv github.com/PeerCodeProject/SignalingServer

exec: build
    serv

run:
    go run cmd/main.go

test:
    go test -v ./server

docker-build:
    docker build -t liquidibrium/signaling-server:latest .

docker-run:
    docker run --name signaling-server -p 4444:4444 liquidibrium/signaling-server

docker-push:
    docker push liquidibrium/signaling-server

sloc:
  wc -l **/*.go