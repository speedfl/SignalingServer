package SignalingServer

import (
	"encoding/json"
	"github.com/fasthttp/websocket"
	"log"
	"net/http"
	"sync"
	"time"
)

const pingTimeout = 30000

const MessageTypeSubscribe = "subscribe"
const MessageTypeUnsubscribe = "unsubscribe"
const MessageTypePublish = "publish"
const MessageTypePing = "ping"
const MessageTypePong = "pong"
const MessageTypeAnnounce = "announce"

type SignalingServer struct {
	serveMux http.ServeMux

	topicsLock sync.Mutex

	//Map from topic-name to set of subscribed clients.
	topics map[string]map[*websocket.Conn]bool

	connectionLocks map[*websocket.Conn]sync.Mutex
}

type Message struct {
	Type   string      `json:"type"`
	Topics []string    `json:"topics"`
	Topic  string      `json:"topic"`
	Data   interface{} `json:"data"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (ss *SignalingServer) handleTest(writer http.ResponseWriter, request *http.Request) {
	log.Println("handleTest:", request.RemoteAddr)
	writer.WriteHeader(http.StatusOK)
	writer.Header().Set("Content-Type", "text/plain")
	_, _ = writer.Write([]byte("OKAY!"))

}

func (ss *SignalingServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ss.serveMux.ServeHTTP(w, r)
}

func (ss *SignalingServer) HandleNewConnection(conn *websocket.Conn) {
	log.Println("New connection:", conn.RemoteAddr())
	closed := false
	subscribedTopics := make(map[string]bool)
	pongReceived := true
	ss.connectionLocks[conn] = sync.Mutex{}
	// send ping every pingTimeout milliseconds
	go func() {
		for !closed {
			if !pongReceived {
				log.Println("Closing connection due to ping timeout:", conn.RemoteAddr())
				err := conn.Close()
				closed = true // anyway change to closed
				if err != nil {
					log.Println("Error closing connection:", err)
					return
				}
			}
			pongReceived = false
			err := conn.SetReadDeadline(time.Now().Add(time.Duration(pingTimeout) * time.Millisecond))
			if err != nil {
				log.Println("Error setting read deadline:", err)
				return
			}
			err = conn.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				log.Println("Error sending ping:", err)
				return
			}
			time.Sleep(time.Duration(pingTimeout) * time.Millisecond)
		}
	}()

	// handle pong
	conn.SetPongHandler(func(appData string) error {
		log.Println("Pong received:", conn.RemoteAddr())
		pongReceived = true
		return nil
	})

	// on close
	conn.SetCloseHandler(func(code int, text string) error {
		closed = true
		log.Println("Closing connection due to close handler:", conn.RemoteAddr())
		ss.topicsLock.Lock()
		defer ss.topicsLock.Unlock()
		for topic := range subscribedTopics {
			subs, ok := ss.topics[topic]
			if !ok {
				log.Println("Error unsubscribing from topic:", topic, ":", conn.RemoteAddr())
				continue
			}
			delete(subs, conn)
			log.Println("Removed connection from topic", topic, ":", conn.RemoteAddr())
			if len(subs) == 0 {
				delete(ss.topics, topic)
			}
			delete(ss.connectionLocks, conn)
		}

		return conn.Close()
	})

	go ss.handleMessage(conn, subscribedTopics)
}

func (ss *SignalingServer) handleMessage(conn *websocket.Conn, subscribedTopics map[string]bool) {
	for {
		msg, err := ss.receiveMessage(conn)
		if err != nil {
			return
		}

		// handle message
		switch msg.Type {
		case MessageTypeSubscribe:
			ss.handleSubscribe(conn, msg, subscribedTopics)
		case MessageTypeUnsubscribe:
			ss.handleUnsubscribe(conn, msg, subscribedTopics)
		case MessageTypePublish:
			ss.handleTopicPublish(msg)
		case MessageTypePing:
			log.Println("Received ping:", conn.RemoteAddr())
			err := ss.sendMessage(conn, &Message{Type: MessageTypePong})
			if err != nil {
				return
			}
		}
	}
}

func (ss *SignalingServer) receiveMessage(conn *websocket.Conn) (*Message, error) {
	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Println("Error reading message:", err)
		return nil, err
	}
	log.Println("Received message:", string(message))

	// parse message
	msg := &Message{}
	err = json.Unmarshal(message, msg)
	if err != nil {
		log.Println("Error parsing message:", err)
		return nil, err
	}
	return msg, nil
}

func (ss *SignalingServer) handleSubscribe(conn *websocket.Conn, msg *Message, subscribedTopics map[string]bool) {
	ss.topicsLock.Lock()
	defer ss.topicsLock.Unlock()
	for _, topicName := range msg.Topics {
		if _, ok := ss.topics[topicName]; !ok {
			ss.topics[topicName] = make(map[*websocket.Conn]bool)
		}
		ss.topics[topicName][conn] = true
		log.Println("Added connection to topic", topicName, ":", conn.RemoteAddr())
		// add topics to subscribedTopics
		subscribedTopics[topicName] = true
	}
}

func (ss *SignalingServer) handleUnsubscribe(conn *websocket.Conn, msg *Message, subscribedTopics map[string]bool) {
	ss.topicsLock.Lock()
	defer ss.topicsLock.Unlock()
	log.Println("Unsubscribing from topics:", msg.Topics)
	for _, topicName := range msg.Topics {
		subs, ok := ss.topics[topicName]
		if ok {
			delete(subs, conn)
			log.Println("Removed connection from topic", topicName, ":", conn.RemoteAddr())
			if len(subs) == 0 {
				delete(ss.topics, topicName)
			}
			delete(subscribedTopics, topicName)
		}
	}
}

func (ss *SignalingServer) handleTopicPublish(msg *Message) {
	ss.topicsLock.Lock()
	defer ss.topicsLock.Unlock()
	log.Println("Publishing message to topic:", msg.Topic)
	receivers, ok := ss.topics[msg.Topic]
	if ok {
		for receiver := range receivers {
			_ = ss.sendMessage(receiver, msg)
		}
	}
}

func (ss *SignalingServer) sendMessage(conn *websocket.Conn, msg *Message) error {
	mutex, ok := ss.connectionLocks[conn]
	if !ok {
		log.Println("Error sending message:", conn.RemoteAddr(), "LOCK not found")
		return nil
	}
	defer mutex.Unlock()
	mutex.Lock()
	// send json message
	err := conn.WriteJSON(msg) // todo make thread safe
	if err != nil {
		log.Println("Error sending message:", err)
		return err
	}
	return nil
}

func NewSignalingServer() *SignalingServer {

	ss := &SignalingServer{
		topics:          make(map[string]map[*websocket.Conn]bool),
		connectionLocks: make(map[*websocket.Conn]sync.Mutex),
	}
	ss.serveMux.HandleFunc("/test", ss.handleTest)
	// upgrade from http to websocket
	ss.serveMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		ss.HandleNewConnection(conn)
	})
	return ss
}
