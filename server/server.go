package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/sirupsen/logrus"
)

const pingTimeout = 30000

type MessageType string

const (
	MessageTypeSubscribe   MessageType = "subscribe"
	MessageTypeUnsubscribe MessageType = "unsubscribe"
	MessageTypePublish     MessageType = "publish"
	MessageTypePing        MessageType = "ping"
	MessageTypePong        MessageType = "pong"
	MessageTypeAnnounce    MessageType = "announce"
)

type SignalingServer struct {
	serveMux http.ServeMux

	topicsLock sync.Mutex

	//Map from topic-name to set of subscribed clients.
	topics map[string]map[*websocket.Conn]bool

	connectionLocks map[*websocket.Conn]*sync.Mutex
}

type Message struct {
	Type   MessageType `json:"type"`
	Topics []string    `json:"topics"`
	Topic  string      `json:"topic"`
	Data   interface{} `json:"data"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (ss *SignalingServer) handleLiveness(writer http.ResponseWriter, request *http.Request) {
	writer.WriteHeader(http.StatusOK)
	writer.Header().Set("Content-Type", "text/plain")
	_, _ = writer.Write([]byte("OK"))

}

func (ss *SignalingServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ss.serveMux.ServeHTTP(w, r)
}

func (ss *SignalingServer) HandleNewConnection(logger *logrus.Entry, conn *websocket.Conn) {
	logger.Info("new connection")
	closed := false
	subscribedTopics := make(map[string]bool)
	pongReceived := true
	ss.connectionLocks[conn] = &sync.Mutex{}
	// send ping every pingTimeout milliseconds
	go func() {
		for !closed {
			if !pongReceived {
				logger.Info("closing connection due to ping timeout")
				err := conn.Close()
				closed = true // anyway change to closed
				if err != nil {
					logger.Warn("error closing connection", err)
					return
				}
			}
			pongReceived = false
			err := conn.SetReadDeadline(time.Now().Add(time.Duration(pingTimeout) * time.Millisecond))
			if err != nil {
				logger.Warn("error setting read deadline", err)
				return
			}
			err = conn.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				logger.Warn("error sending ping", err)
				return
			}
			time.Sleep(time.Duration(pingTimeout) * time.Millisecond)
		}
	}()

	// handle pong
	conn.SetPongHandler(func(appData string) error {
		logger.Info("pong received")
		pongReceived = true
		return nil
	})

	// on close
	conn.SetCloseHandler(func(code int, text string) error {
		closed = true
		logger.Info("closing connection due to close handler")
		ss.topicsLock.Lock()
		defer ss.topicsLock.Unlock()
		for topic := range subscribedTopics {
			subs, ok := ss.topics[topic]
			if !ok {
				logger.Infof(`error unsubscribing from topic "%s"`, topic)
				continue
			}
			delete(subs, conn)
			logger.Infof(`removed connection from topic "%s"`, topic)
			if len(subs) == 0 {
				delete(ss.topics, topic)
			}
			delete(ss.connectionLocks, conn)
		}

		return conn.Close()
	})

	go ss.handleMessage(logger, conn, subscribedTopics)
}

func (ss *SignalingServer) handleMessage(logger *logrus.Entry, conn *websocket.Conn, subscribedTopics map[string]bool) {
	for {
		msg, err := ss.receiveMessage(logger, conn)
		if err != nil {
			return
		}

		// handle message
		switch msg.Type {
		case MessageTypeSubscribe:
			ss.handleSubscribe(logger, conn, msg, subscribedTopics)
		case MessageTypeUnsubscribe:
			ss.handleUnsubscribe(logger, conn, msg, subscribedTopics)
		case MessageTypePublish:
			ss.handleTopicPublish(logger, msg)
		case MessageTypePing:
			logger.Info("received ping")
			err := ss.sendMessage(logger, conn, &Message{Type: MessageTypePong})
			if err != nil {
				return
			}
		}
	}
}

func (ss *SignalingServer) receiveMessage(logger *logrus.Entry, conn *websocket.Conn) (*Message, error) {
	_, message, err := conn.ReadMessage()
	if err != nil {
		logger.Warn("error reading message", err)
		return nil, fmt.Errorf("error reading message, %w", err)
	}

	logger.Debug("received message:", string(message))

	// parse message
	msg := &Message{}
	err = json.Unmarshal(message, msg)
	if err != nil {
		logger.Warn("error parsing message", err)
		return nil, fmt.Errorf("error parsing message, %w", err)
	}
	return msg, nil
}

func (ss *SignalingServer) handleSubscribe(logger *logrus.Entry, conn *websocket.Conn, msg *Message, subscribedTopics map[string]bool) {
	ss.topicsLock.Lock()
	defer ss.topicsLock.Unlock()
	for _, topicName := range msg.Topics {
		if _, ok := ss.topics[topicName]; !ok {
			ss.topics[topicName] = make(map[*websocket.Conn]bool)
		}
		ss.topics[topicName][conn] = true
		logger.Infof(`added connection to topic "%s"`, topicName)
		// add topics to subscribedTopics
		subscribedTopics[topicName] = true
	}
}

func (ss *SignalingServer) handleUnsubscribe(logger *logrus.Entry, conn *websocket.Conn, msg *Message, subscribedTopics map[string]bool) {
	ss.topicsLock.Lock()
	defer ss.topicsLock.Unlock()
	logger.Infof("unsubscribing from topics %s", strings.Join(msg.Topics, ","))
	for _, topicName := range msg.Topics {
		subs, ok := ss.topics[topicName]
		if ok {
			delete(subs, conn)
			logger.Infof(`removed connection from topic "%s"`, topicName)
			if len(subs) == 0 {
				delete(ss.topics, topicName)
			}
			delete(subscribedTopics, topicName)
		}
	}
}

func (ss *SignalingServer) handleTopicPublish(logger *logrus.Entry, msg *Message) {
	ss.topicsLock.Lock()
	defer ss.topicsLock.Unlock()
	logger.Infof(`publishing message to topic "%s"`, msg.Topic)
	receivers, ok := ss.topics[msg.Topic]
	if ok {
		for receiver := range receivers {
			_ = ss.sendMessage(logger, receiver, msg)
		}
	}
}

func (ss *SignalingServer) sendMessage(logger *logrus.Entry, conn *websocket.Conn, msg *Message) error {
	mutex, ok := ss.connectionLocks[conn]
	if !ok {
		logger.Error("error sending message, lock not found")
		return nil
	}
	defer mutex.Unlock()
	mutex.Lock()
	// send json message
	// todo make thread safe
	if err := conn.WriteJSON(msg); err != nil {
		logger.Warn("error sending message", err)
		return fmt.Errorf("error sending message, %w", err)
	}

	return nil
}

func (ss *SignalingServer) getTopics() (topics []string) {
	ss.topicsLock.Lock()
	defer ss.topicsLock.Unlock()
	for topic := range ss.topics {
		topics = append(topics, topic)
	}
	return
}

func NewSignalingServer() *SignalingServer {

	ss := &SignalingServer{
		topics:          make(map[string]map[*websocket.Conn]bool),
		connectionLocks: make(map[*websocket.Conn]*sync.Mutex),
	}

	ss.serveMux.HandleFunc("/health/live", ss.handleLiveness)
	// upgrade from http to websocket
	ss.serveMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		// TODO: Auth

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		logger := logrus.WithField("address", conn.RemoteAddr())
		ss.HandleNewConnection(logger, conn)
	})
	return ss
}

func RunServer(addr string) (err error, serv *http.Server) {
	ss := NewSignalingServer()
	serv = &http.Server{
		Addr:         addr,
		Handler:      ss,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}
	err = serv.ListenAndServe()
	return
}
