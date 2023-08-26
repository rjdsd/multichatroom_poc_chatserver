package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type ChatServer struct {
	ChatRooms   map[string]*ChatRoom
	Upgrader    websocket.Upgrader
	ClientConns map[string]*websocket.Conn
}

func InitChatServer() *ChatServer {
	chatServer := ChatServer{
		ChatRooms:   make(map[string]*ChatRoom),
		ClientConns: make(map[string]*websocket.Conn),
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
	// for now this is hardcoded. we can create feature to create new ChatRoom dynamically
	sports := InitChatRoom("sports")
	travel := InitChatRoom("travel")
	chatServer.ChatRooms["sports"] = sports
	chatServer.ChatRooms["travel"] = travel
	return &chatServer
}

func InitChatRoom(name string) *ChatRoom {
	chatRoom := ChatRoom{
		ChatRoomName: name,
		ClientConns:  make(map[*websocket.Conn]bool),
	}
	return &chatRoom
}

type ChatRoom struct {
	ChatRoomName string
	ClientConns  map[*websocket.Conn]bool
}

type JoinChatRoom struct {
	ChatRoomName string `json:"chatroomname"`
	ClientID     string `json:"clientid"`
}

type ChatMessage struct {
	ChatRoom string `json:"chatroom"`
	Username string `json:"username"`
	Text     string `json:"text"`
}

type ChatClient struct {
	ClientName string `json:"clientname"`
}

func (chatServer *ChatServer) Cleanup() {
	fmt.Println("Cleaning up")
	for _, chatRoom := range chatServer.ChatRooms {
		for ws := range chatRoom.ClientConns {
			ws.Close()
		}
	}
}

func (chatServer *ChatServer) ClientConnectHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("request received to add client to ChatServer")
	ws, err := chatServer.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("can not join chatroom", err)
		return
	}
	clientId := uuid.New().String()
	chatServer.ClientConns[clientId] = ws
	ws.WriteMessage(websocket.TextMessage, []byte(clientId))
	fmt.Println("joined ChatServer, ClientID:", clientId)
}

func (chatServer *ChatServer) JoinChatRoomHandler(w http.ResponseWriter, r *http.Request) {
	allowCORS(w)
	var joinRoom JoinChatRoom
	err := json.NewDecoder(r.Body).Decode(&joinRoom)
	if err != nil {
		fmt.Println("Error decoding packet, dropping", err)
		return
	}
	if joinRoom.ClientID == "" || joinRoom.ChatRoomName == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("ClientID or ChatRoomName is missing"))
	}
	fmt.Println("received request to add chat user to ChatRoom:", joinRoom.ChatRoomName, joinRoom.ClientID)
	chatRoom, ok := chatServer.ChatRooms[joinRoom.ChatRoomName]
	if !ok {
		fmt.Println("ChatRoom Doesn't Exist:", joinRoom.ChatRoomName)
		return
	}

	connection, isValid := chatServer.ClientConns[joinRoom.ClientID]
	if !isValid {
		fmt.Println("client Doesn't Exist:")
		return
	}
	chatRoom.ClientConns[connection] = true

	var chatMsg ChatMessage
	chatMsg.ChatRoom = joinRoom.ChatRoomName
	chatMsg.Username = "ChatServer"
	chatMsg.Text = "new user joined chatroom"
	chatServer.SendMessageToMembers(&chatMsg)

	fmt.Println("chat user could joined ChatRoom:", joinRoom.ChatRoomName, chatRoom.ClientConns)
}

func (chatServer *ChatServer) SendMessageHandler(w http.ResponseWriter, r *http.Request) {
	allowCORS(w)
	var chatMsg ChatMessage
	err := json.NewDecoder(r.Body).Decode(&chatMsg)
	if err != nil {
		fmt.Println("Error decoding empty packet, dropping", err)
		return
	}
	chatServer.SendMessageToMembers(&chatMsg)
}

func (chatServer *ChatServer) SendMessageToMembers(chatMsg *ChatMessage) {
	chatRoom, ok := chatServer.ChatRooms[chatMsg.ChatRoom]
	if !ok {
		fmt.Println("ChatRoom Doesn't Exist:", chatMsg.ChatRoom)
		return
	}
	memberConns := chatRoom.ClientConns
	msg := chatMsg.Username + ":" + chatMsg.Text
	for wsClientCon, connectionAlive := range memberConns {
		if connectionAlive {
			wsClientCon.WriteMessage(websocket.TextMessage, []byte(msg))
		}
	}
}

func allowCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST,GET,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
}
