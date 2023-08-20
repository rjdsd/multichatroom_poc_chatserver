package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	chatServer := InitChatServer()
	http.HandleFunc("/joinChatServer", chatServer.ClientConnectHandler)
	http.HandleFunc("/joinChatRoom", chatServer.JoinChatRoomHandler)
	http.HandleFunc("/sendMsg", chatServer.SendMessageHandler)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		chatServer.Cleanup()
		os.Exit(1)
	}()

	http.ListenAndServe(":8086", nil)
}
