package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"os"
	"time"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:    1024,
		WriteBufferSize:   4096,
		EnableCompression: true,
	}
)

func serveWs(w http.ResponseWriter, r *http.Request, user string, userid string, path string, msg int) {
	var (
		lastcontent string
		ufile       *os.File
	)

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure)
		c.Close()
		return
	}

	c.SetReadLimit(512)
	c.SetPongHandler(func(string) error { c.SetReadDeadline(time.Now().Add(15 * time.Second)); return nil })
	c.EnableWriteCompression(true)
	c.SetCompressionLevel(9)
	done := make(chan struct{})
	pingTicker := time.NewTicker(5 * time.Second)
	fileTicker := time.NewTicker(time.Duration(msg*60) * time.Millisecond)

	defer close(done)

	go func() {
		for {
			select {
			case <-fileTicker.C:
				content := getPostContent(file, msg)
				if content != lastcontent {
					c.SetWriteDeadline(time.Now().Add(10 * time.Second))
					c.WriteMessage(websocket.TextMessage, []byte(content))
					lastcontent = content
				}
			case <-pingTicker.C:
				c.SetWriteDeadline(time.Now().Add(10 * time.Second))
				c.WriteMessage(websocket.PingMessage, []byte{})
			case <-done:
				websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure)
				c.Close()
				file.Close()
				ufile.Close()
				fileTicker.Stop()
				pingTicker.Stop()
				fmt.Println("User " + userid + " has disconnected from " + path[12:len(path)-4])
				return
			}
		}
	}()

	for {
		_, messageraw, err := c.ReadMessage()
		if err != nil {
			websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure)
			break
		}

		writeMsg(file, userid, user, messageraw)
	}
}
