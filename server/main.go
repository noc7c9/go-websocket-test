package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

// Web server
func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/connect", connect)

	// Web API for sending message
	mux.HandleFunc("/api/broadcast", webApiBroadcast)
	mux.HandleFunc("/api/list", webApiList)
	mux.HandleFunc("/api/send/", webApiSend)

	handler := cors.Default().Handler(mux)

	port, ok := os.LookupEnv("PORT")
	var addr string
	if ok {
		addr = ":" + port
	} else {
		addr = ":3000"
	}

	fmt.Println("Starting server on", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}

// Outgoing messages helpers
func Msg(type_ string) map[string]interface{} {
	msg := make(map[string]interface{})
	msg["type"] = type_
	return msg
}
func MsgPong() map[string]interface{} {
	return Msg("PONG")
}
func MsgAssignId(id uuid.UUID) map[string]interface{} {
	msg := Msg("ASSIGN_ID")
	msg["id"] = id
	return msg
}
func MsgConnected(id uuid.UUID) map[string]interface{} {
	msg := Msg("CONNECTED")
	msg["id"] = id
	return msg
}
func MsgDisconnected(id uuid.UUID) map[string]interface{} {
	msg := Msg("DISCONNECTED")
	msg["id"] = id
	return msg
}
func MsgText(sender uuid.UUID, text string) map[string]interface{} {
	msg := Msg("TEXT")
	msg["sender"] = sender
	msg["text"] = text
	return msg
}

// WebSocket handlers

// Keep track of all connections for broadcasting
var conns = make(map[uuid.UUID]chan interface{})

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	// UNSAFE: let any connection through
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Handler for making a WebSocket connection
func connect(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP call into a WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// Generate an ID for the user
	id := uuid.Must(uuid.NewV4())
	fmt.Println(id, "Connected")

	// Create a channel for the user
	ch := make(chan interface{})
	conns[id] = ch

	go writer(id, conn, ch)
	go reader(id, conn, ch)

	// Send ID to user
	ch <- MsgAssignId(id)

	// Tell other users about new user
	broadcast(id, MsgConnected(id))
}

// Reads incoming WebSocket messages
func reader(id uuid.UUID, conn *websocket.Conn, ch chan interface{}) {
	for {
		// Wait for a message
		_, bytes, err := conn.ReadMessage()
		if err != nil {
			// Unable to read, so user disconnected
			fmt.Println(id, "Disconnected")

			delete(conns, id) // Remove from map of connections
			close(ch)         // Stop the writer

			// Tell other users about user disconnecting
			broadcast(id, MsgDisconnected(id))
			return
		}

		// Figure out message type
		var msg map[string]interface{}
		if err := json.Unmarshal(bytes, &msg); err != nil {
			log.Println(id, "Unable to deserialize message:", bytes)
			continue
		}

		fmt.Println(id, "Received", msg)
		switch msg["type"] {
		case "PING":
			ch <- MsgPong()
		case "TEXT":
			text, ok := msg["text"].(string)
			if !ok {
				log.Println(id, "Invalid TEXT message:", msg)
				continue
			}

			// Send text to other users
			broadcast(id, MsgText(id, text))
		}
	}
}

// Writes outcoming WebSocket messages
func writer(id uuid.UUID, conn *websocket.Conn, ch chan interface{}) {
	for {
		// Wait for a message to send
		msg := <-ch

		// Channel has been closed, meaning the user disconnected
		if msg == nil {
			return
		}

		fmt.Println(id, "Sending", msg)
		if err := conn.WriteJSON(msg); err != nil {
			log.Println(err)
			return
		}
	}
}

// Send a message to all connected users excluding the sender
func broadcast(senderId uuid.UUID, msg interface{}) {
	fmt.Println(senderId, "Broadcasting", msg)
	for id, ch := range conns {
		// Don't send message to the sender
		if id == senderId {
			continue
		}

		fmt.Println(id, "BC-Sending", msg)
		ch <- msg
	}
}

// Web API handlers

func webApiBroadcast(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(405)
		fmt.Fprintln(w, "405 method not allowed")
		return
	}

	fmt.Println("Web API (Broadcast) called")

	// read the body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Println("Failed to read request body")
		return
	}

	// ensure the body is valid JSON
	var msg map[string]interface{}
	if err := json.Unmarshal(body, &msg); err != nil {
		w.WriteHeader(400)
		fmt.Fprintln(w, "400 body is not valid json")
		return
	}

	// broadcast to all users
	broadcast(uuid.Nil, msg)

	w.WriteHeader(200)
	fmt.Fprintln(w, "200 ok")
	return
}

func webApiList(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(405)
		fmt.Fprintln(w, "405 method not allowed")
		return
	}

	fmt.Println("Web API (List) called")

	w.WriteHeader(200)
	fmt.Fprintln(w, len(conns), "connected users")
	for id := range conns {
		fmt.Fprintln(w, id)
	}
	return
}
func webApiSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(405)
		fmt.Fprintln(w, "405 method not allowed")
		return
	}

	fmt.Println("Web API (Send) called")

	// parse the id from the path
	idStr := strings.Replace(r.URL.Path, "/api/send/", "", 1)
	id := uuid.FromStringOrNil(idStr)

	ch, ok := conns[id]

	if !ok {
		w.WriteHeader(400)
		fmt.Fprintln(w, "400 unknown user id")
		return
	}

	// read the body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Println("Failed to read request body")
		return
	}

	// ensure the body is valid JSON
	var msg map[string]interface{}
	if err := json.Unmarshal(body, &msg); err != nil {
		w.WriteHeader(400)
		fmt.Fprintln(w, "400 body is not valid json")
		return
	}

	// send the message to the specified user
	ch <- msg

	w.WriteHeader(200)
	fmt.Fprintln(w, "200 ok")
	return
}
