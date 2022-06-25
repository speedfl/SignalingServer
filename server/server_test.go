package server

import (
	"encoding/json"
	"github.com/PeerCodeProject/SignalingServer/utils"
	"github.com/fasthttp/websocket"
	"io"
	"log"
	"net/http"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestNewSignalingServer(t *testing.T) {

	t.Run("test ss", func(t *testing.T) {
		// run server
		ss := NewSignalingServer()
		addr := utils.GetPort()
		serv := &http.Server{
			Addr:         addr,
			Handler:      ss,
			ReadTimeout:  time.Second * 10,
			WriteTimeout: time.Second * 10,
		}
		defer serv.Close()
		log.Println("Listening on " + addr)
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			wg.Done()
			_ = serv.ListenAndServe()
		}()

		wg.Wait()
		// make http request to /test
		resp, err := http.Get("http://localhost" + addr + "/test")
		if err != nil {
			t.Errorf("Error http GET: %v", err)
		}
		bytes := make([]byte, 1024)
		read, err := resp.Body.Read(bytes)
		if err != nil && err != io.EOF {
			t.Errorf("Error reading body: %v", err)
		}

		if !reflect.DeepEqual(resp.StatusCode, http.StatusOK) {
			t.Errorf("receiveMessage() got = %v, want %v", resp.Status, http.StatusOK)
		}
		text := string(bytes[:read])
		expected := "OKAY!"
		if !reflect.DeepEqual(text, expected) {
			t.Errorf("receiveMessage() got = %v, want %v", text, expected)
		}
	})

}

func TestWS(t *testing.T) {

	t.Run("test ws", func(t *testing.T) {
		// run server
		ss := NewSignalingServer()
		addr := utils.GetPort()
		serv := &http.Server{
			Addr:         addr,
			Handler:      ss,
			ReadTimeout:  time.Second * 10,
			WriteTimeout: time.Second * 10,
		}
		defer serv.Close()

		log.Println("Listening on " + addr)
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			wg.Done()
			_ = serv.ListenAndServe()
		}()

		wg.Wait()
		address := "ws://localhost" + addr
		c1, err := GetClient(address)

		topic1 := "ROOM"
		err = c1.WriteJSON(Message{Type: MessageTypeSubscribe, Topics: []string{topic1}})
		if err != nil {
			return
		}
		err = c1.WriteJSON(Message{Type: MessageTypeSubscribe, Topics: []string{"topic2"}})
		if err != nil {
			return
		}

		// wait for messages to be received for 5 sec
		log.Println("sleeping for 5 sec")
		time.Sleep(5 * time.Second)

		topics := ss.getTopics()
		if len(topics) != 2 {
			t.Errorf("MessageTypeSubscribe topics got: %v", topics)
		}
		data := "hello"
		c2, err := GetClient(address)
		if err != nil {
			return
		}
		c2.WriteJSON(Message{Type: MessageTypeSubscribe, Topics: []string{topic1}})
		log.Println("sleeping for 5 sec")
		time.Sleep(5 * time.Second)

		go func() {

			message, p, err := c2.ReadMessage()
			if err != nil {
				return
			}
			if message != websocket.TextMessage {
				t.Errorf("MessageTypeSubscribe message got: %v", message)
			}
			if p == nil {
				t.Errorf("MessageTypeSubscribe p got: %v", p)
			}
			var m Message
			err = json.Unmarshal(p, &m)
			if err != nil {
				t.Errorf("MessageTypeSubscribe json.Unmarshal got: %v", err)
			}
			if m.Type != MessageTypePublish {
				t.Errorf("MessageTypeSubscribe message got: %v", m.Type)
			}
			log.Println("Message m", m)
			if !reflect.DeepEqual(m.Topic, topic1) {
				t.Errorf("MessageTypeSubscribe message topic got: %v", m.Topic)
			}
			if !reflect.DeepEqual(m.Data, data) {
				t.Errorf("MessageTypeSubscribe message data got: %v", m.Data)
			}
			wg.Done()
		}()
		wg.Add(1)
		log.Println("publishing message")
		c1.WriteJSON(Message{Type: MessageTypePublish, Topic: topic1, Data: data})

		wg.Wait()
		//time.Sleep(5 * time.Second)

	})

}

func GetClient(address string) (*websocket.Conn, error) {
	c, _, err := websocket.DefaultDialer.Dial(address, nil)
	c.SetPingHandler(
		func(appData string) error {
			log.Println("ping", appData)
			return nil
		})
	c.SetPongHandler(func(appData string) error {
		log.Println("pong", appData)
		return nil
	})
	return c, err
}
