package main

import (
	"crypto/rand"
	"encoding/base64"
	"github.com/serverhorror/rog-go/reverse"
	"io/ioutil"
	"mvdan.cc/xurls"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func getPostContent(file *os.File, msg int) string {
	var (
		header string
		post   string
		i      int
	)

	// Shrink files larger than 10kb
	fi, err := file.Stat()
	if err == nil && fi.Size() > 10000 {
		input, err := ioutil.ReadFile("data/boards/" + fi.Name())
		if err != nil {
			return `<div class="post" id="post"><h4>Error : Unable to read board posts.</h4><p>Please try again later.</p></div><br>`
		}

		output := input[200:]

		ioutil.WriteFile("data/boards/"+fi.Name(), output, 0666)
	}

	scan := reverse.NewScanner(file)
	for scan.Scan() {
		text := scan.Text()
		postarray := strings.Split(text, "<")

		if len(postarray) != 4 {
			break
		}

		header = postarray[0]
		if len(postarray[1]) > 2 {
			header = postarray[0] + " (" + postarray[1] + ")"
		}

		message := xurls.Strict().ReplaceAllStringFunc(postarray[2], func(data string) string {
			u, err := url.Parse(data)
			if err != nil {
				return `<a href="` + data + `">` + data + "</a>"
			}

			if strings.HasSuffix(u.EscapedPath(), ".png") || strings.HasSuffix(u.EscapedPath(), ".jpg") || strings.HasSuffix(u.EscapedPath(), ".webp") || strings.HasSuffix(u.EscapedPath(), ".jpeg") || strings.HasSuffix(u.EscapedPath(), ".gif") {
				return `<img src="` + data + `" title="Image from ` + u.Host + `"></img>`
			}

			if strings.HasSuffix(u.EscapedPath(), ".mp3") || strings.HasSuffix(u.EscapedPath(), ".ogg") || strings.HasSuffix(u.EscapedPath(), ".wav") {
				return `<audio controls src="` + data + `"></audio>`
			}

			if strings.HasSuffix(u.EscapedPath(), ".mp4") || strings.HasSuffix(u.EscapedPath(), ".webm") || strings.HasSuffix(u.EscapedPath(), ".ogg") {
				return `<video controls src="` + data + `"></video>`
			}

			return `<a href="` + data + `">` + u.Host + u.EscapedPath() + "</a>"
		})

		tstamp := postarray[3]
		t, err := time.Parse(time.UnixDate, postarray[3])
		if err == nil {
			if time.Since(t).Hours() > 1 {
				tstamp = strconv.FormatFloat(time.Since(t).Hours(), 'f', 0, 64) + " hours ago"
			} else if time.Since(t).Minutes() > 1 {
				tstamp = strconv.FormatFloat(time.Since(t).Minutes(), 'f', 0, 64) + " minutes ago"
			} else {
				tstamp = "<1 minute ago"
			}
		}

		post = post + `<div class="post" id="post"><p class="time">` + tstamp + `</p><h4>` + header + `</h4><p>` + message + `</p></div><br>` + "\n"
		i = i + 1
		if i > msg {
			break
		}
	}
	return post
}

func checkBoard(board string) string {
	if board == "" || strings.ContainsAny(board, ".") {
		return "data/boards/main.txt"
	} else {
		_, err := os.Stat("data/boards/" + board + ".txt")
		if err != nil {
			return "data/boards/main.txt"
		}

		return "data/boards/" + board + ".txt"
	}
}

func newBoard(w http.ResponseWriter, r *http.Request, userid string) {
	rand, err := generateRandomBytes(20)
	if err != nil {
		http.Error(w, "500 Internal Server Error : An unexpected condition was encountered.", http.StatusInternalServerError)
		return
	}

	new := base64.RawURLEncoding.EncodeToString(rand)
	f, err := os.Create("data/boards/" + new + ".txt")
	if err != nil {
		http.Error(w, "500 Internal Server Error : An unexpected condition was encountered.", http.StatusInternalServerError)
		return
	}

	f.WriteString("KatBoard<<Welcome to a new board, created by " + userid + "! You can invite more people to your new board by sharing the link!<" + time.Now().UTC().Format(time.UnixDate) + "\n")
	f.WriteString("KatBoard<<Note : KatBoard is currently a work in progress, so boards may be cleared or removed at any time.<\n")
	f.Close()

	http.Redirect(w, r, "/?board="+new, http.StatusTemporaryRedirect)
}
