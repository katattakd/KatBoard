// KatBoard Beta - An image-board like social media network
package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"
)

var banList []string

func runAuth(w http.ResponseWriter, r *http.Request) (string, string) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Please enter a password and/or a nickname. Set your password to anon to post anonymously.`)
	user, pass, _ := r.BasicAuth()
	if pass == "" && len(user) <= 64 {
		http.Error(w, `Please enter a password and/or a nickname. Set your password to anon to post anonymously.`, http.StatusUnauthorized)
		return "Error", ""
	}

	if pass == "anon" {
		return user, "Anonymous"
	}

	h := sha256.New()
	h.Write([]byte(user + pass))
	userid := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	return user, userid
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
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
	}

	i := sort.SearchStrings(banList, ip)
	if i < len(banList) && banList[i] == ip {
		http.Error(w, "You have been banned from KatBoard.", http.StatusForbidden)
	}

	user, userid := runAuth(w, r)
	if user == "Error" {
		return
	}

	addHeaders(w, r)
	fmt.Println("Request (" + ip + " - " + userid + ") : " + r.URL.String())

	switch {
	case strings.HasSuffix(r.URL.EscapedPath(), "/messagesocket"):
		msg, err := strconv.Atoi(r.URL.Query().Get("posts"))
		if err != nil || msg <= 1 || msg > 100 {
			msg = 8
		}

		serveWs(w, r, template.HTMLEscapeString(user), userid, checkBoard(r.URL.Query().Get("board")), msg-1)
	case strings.HasSuffix(r.URL.EscapedPath(), "/newboard"):
		w.Header().Set("Cache-Control", "no-store, must-revalidate")
		newBoard(w, r, userid)
	default:
		serveFile(w, r, "data/index.html")
		w.Header().Set("Cache-Control", "max-age=172800, public, stale-while-revalidate=86400")
		w.Header().Set("Expires", time.Now().UTC().Add(24*time.Hour).Format(http.TimeFormat))
	}
}

func main() {
	content, err := ioutil.ReadFile("data/iplist.txt")
	if err == nil {
		banList = strings.Split(string(content), "\n")
		sort.Strings(banList)
	}
	debug.SetGCPercent(600)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      http.HandlerFunc(mainHandle),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	srv.ListenAndServe()
}
