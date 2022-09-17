# Signaling Server

WebRTC Signaling Server in Go for [y-webrtc](https://github.com/yjs/y-webrtc/blob/master/bin/server.js)

## Usage

bash

```bash
    go run cmd/main.go
```

Docker

```bash
    docker run --name signaling-server -p 4444:4444 liquidibrium/signaling-server
```

More comands in `justfile`

## References

* Client library - [simple peer](https://github.com/feross/simple-peer)
* Based on - [fasthttp WebSocket](https://github.com/fasthttp/websocket)
