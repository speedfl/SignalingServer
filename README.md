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

## Testing

To test you cna install a tool such as [wscat](https://github.com/websockets/wscat):

```
npm install -g wscat
```

Then open two terminals:

- Terminal 1:

```
wscat -c ws://localhost:4444
Connected (press CTRL+C to quit)
> {"type":"subscribe","topics":["golang"]}
```

- Terminal 2:

```
wscat -c ws://localhost:4444
Connected (press CTRL+C to quit)
> {"type":"subscribe","topics":["golang"]}
> {"type":"publish","topic":"golang","data":{"message":"New Go update released!!!","version":"1.19","url":"https://golang.org/dl/"}}
```

You should then see in terminal 1:

```
< {"type":"publish","topics":null,"topic":"golang","data":{"message":"New Go update released!!!","url":"https://golang.org/dl/","version":"1.19"}}
```

## References

* Client library - [simple peer](https://github.com/feross/simple-peer)
* Based on - [fasthttp WebSocket](https://github.com/fasthttp/websocket)

