// KatBoard Beta - An image-board like social media network
package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

func runAuth(w http.ResponseWriter, r *http.Request) (string, string, string) {
	var user, pass string
	user, pass, _ = r.BasicAuth()

	w.Header().Set("WWW-Authenticate", `Basic realm="Please enter a password to use as your login, nicknames are optional. Set your password to anon to post anonymously.`)

	v := r.URL.Query()
	if u, ok := v["user"]; ok && len(u) > 0 {
		user = u[0]
	}
	if p, ok := v["pass"]; ok && len(p) > 0 {
		pass = p[0]
	}

	if pass == "" || len(user) > 64 {
		http.Error(w, `Please enter a password to use as your login, nicknames are optional. Set your password to anon to post anonymously.`, http.StatusUnauthorized)
		return "Error", "", ""
	}

	if user == "anon" {
		user = ""
	}

	if pass == "anon" {
		return user, "Anonymous", "anon"
	}

	h := sha256.New()
	h.Write([]byte(user + pass))
	userid := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	return user, userid, pass
}

func serveFile(w http.ResponseWriter, r *http.Request, loc string) error {
	file, err := os.Open(loc)
	if err != nil {
		return err
	}

	finfo, err := file.Stat()
	if err != nil {
		return err
	}

	if strings.Contains(r.Header.Get("Accept-Encoding"), "br") {
		if filen, err := os.Open(loc + ".br"); err == nil {
			file.Close()
			file = filen
			w.Header().Set("Content-Encoding", "br")
		}
	}

	http.ServeContent(w, r, finfo.Name(), finfo.ModTime(), file)
	return file.Close()
}

func addHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Server", "KatBoard (KatWeb based)")
	w.Header().Add("Referrer-Policy", "no-referrer")
	w.Header().Add("X-Content-Type-Options", "nosniff")
	w.Header().Add("X-Frame-Options", "sameorigin")
	w.Header().Add("X-XSS-Protection", "1; mode=block")
}

func mainHandle(w http.ResponseWriter, r *http.Request) {
	user, userid, pass := runAuth(w, r)
	if user == "Error" {
		return
	}

	addHeaders(w, r)

	switch {
	case strings.HasSuffix(r.URL.EscapedPath(), "/messagesocket"):
		msg, err := strconv.Atoi(r.URL.Query().Get("posts"))
		if err != nil || msg <= 1 || msg > 100 {
			msg = 8
		}

		board := checkBoard(r.URL.Query().Get("board"))
		fmt.Println("User " + userid + " has connected to " + board[12:len(board)-4])
		serveWs(w, r, template.HTMLEscapeString(user), userid, board, msg-1)
	case strings.HasSuffix(r.URL.EscapedPath(), "/newboard"):
		w.Header().Set("Cache-Control", "no-store, must-revalidate")
		newBoard(w, r, userid)
		fmt.Println("User " + userid + " has created a new board.")
	case strings.HasSuffix(r.URL.EscapedPath(), "/getboardmsgs"):
		w.Header().Set("Cache-Control", "no-store, must-revalidate")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		f, err := os.Open(checkBoard(r.URL.Query().Get("board")))
		if err != nil {
			io.WriteString(w, `<div id="post"><h4>Error : Unable to read board posts.</h4><p>Please try again later.</p></div><br>`)
			return
		}
		io.WriteString(w, getPostContent(f, 4))
		f.Close()
	case strings.HasSuffix(r.URL.EscapedPath(), "/sendboardmsg"):
		w.Header().Set("Cache-Control", "no-store, must-revalidate")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		f, err := os.OpenFile(checkBoard(r.URL.Query().Get("board")), os.O_APPEND|os.O_RDWR, 0600)
		if err != nil {
			io.WriteString(w, "Unable to send message!")
			return
		}
		writeMsg(f, userid, user, []byte(r.URL.Query().Get("msg")))
		f.Close()
		io.WriteString(w, "Message sent.")
	case strings.HasSuffix(r.URL.EscapedPath(), "/sendboardmsgif"):
		w.Header().Set("Cache-Control", "no-store, must-revalidate")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, `<form action="/sendboardmsgif" method="get"><input name="board" autocomplete="on" placeholder="Board ID" type="text"><input name="msg" autocomplete="off" placeholder="Type something to post..." type="text">
		<input type="submit" value="Send Message"></form>`)
		f, err := os.OpenFile(checkBoard(r.URL.Query().Get("board")), os.O_APPEND|os.O_RDWR, 0600)
		if err != nil {
			return
		}
		writeMsg(f, userid, user, []byte(r.URL.Query().Get("msg")))
		f.Close()
	case strings.HasSuffix(r.URL.EscapedPath(), "/mylogin"):
		w.Header().Set("Cache-Control", "no-store, must-revalidate")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, "&user="+user+"&pass="+pass)
	case strings.HasSuffix(r.URL.EscapedPath(), "/favicon.ico"):
		w.WriteHeader(http.StatusNotFound)
		return
	default:
		serveFile(w, r, "data/index.html")
		w.Header().Set("Cache-Control", "max-age=172800, public, stale-while-revalidate=86400")
		w.Header().Set("Expires", time.Now().UTC().Add(24*time.Hour).Format(http.TimeFormat))
	}
}

func main() {
	debug.SetGCPercent(600)

	srv := &http.Server{
		Addr:         "localhost:8081",
		Handler:      http.HandlerFunc(mainHandle),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	srv.ListenAndServe()
}
