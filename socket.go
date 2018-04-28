package main

import (
	"bytes"
	"github.com/gorilla/websocket"
	"html/template"
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
	c.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.SetPongHandler(func(string) error { c.SetReadDeadline(time.Now().Add(60 * time.Second)); return nil })
	c.EnableWriteCompression(true)
	c.SetCompressionLevel(9)
	pingTicker := time.NewTicker(54 * time.Second)
	fileTicker := time.NewTicker(time.Duration(msg*60) * time.Millisecond)

	defer func() {
		c.Close()
		pingTicker.Stop()
		file.Close()
		ufile.Close()
	}()

	go func() {
		for {
			select {
			case <-fileTicker.C:
				content := getPostContent(file, msg)
				if content != lastcontent {
					c.SetWriteDeadline(time.Now().Add(10 * time.Second))
					if err := c.WriteMessage(websocket.TextMessage, []byte(content)); err != nil {
						return
					}

					lastcontent = content
				}
			case <-pingTicker.C:
				c.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := c.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
					return
				}
			}
		}
	}()

	for {
		_, messageraw, err := c.ReadMessage()
		if err != nil {
			websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure)
			break
		}

		message := template.HTMLEscapeString(string(bytes.TrimSpace(bytes.Replace(messageraw, []byte{'\n'}, []byte{' '}, -1))))
		tstamp := time.Now().UTC().Format(time.UnixDate)
		if len(message) <= 512 && len(message) > 1 && len(user) <= 64 {
			file.WriteString(userid + "<" + user + "<" + message + "<" + tstamp + "\n")
		}
		time.Sleep(100 * time.Millisecond)
	}
}
