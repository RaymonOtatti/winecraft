// Command cdpshot saves a screenshot of the page open in a Chrome started with
// --remote-debugging-port, using the DevTools protocol. It exists to test the
// browser client end to end (BUILD.md B5.3) without a person at the screen.
//
//	cdpshot -port 9301 -out snap/a.png
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/coder/websocket"
)

func main() {
	port := flag.Int("port", 9222, "Chrome remote debugging port")
	out := flag.String("out", "shot.png", "PNG file to write")
	flag.Parse()

	res, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/json", *port))
	if err != nil {
		log.Fatal(err)
	}
	var targets []struct {
		Type, URL, WebSocketDebuggerURL string
	}
	if err := json.NewDecoder(res.Body).Decode(&targets); err != nil {
		log.Fatal(err)
	}
	res.Body.Close()
	ws := ""
	for _, t := range targets {
		if t.Type == "page" {
			ws = t.WebSocketDebuggerURL
			break
		}
	}
	if ws == "" {
		log.Fatal("no page target")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, ws, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.CloseNow()
	conn.SetReadLimit(64 << 20)

	req, _ := json.Marshal(map[string]any{"id": 1, "method": "Page.captureScreenshot", "params": map[string]string{"format": "png"}})
	if err := conn.Write(ctx, websocket.MessageText, req); err != nil {
		log.Fatal(err)
	}
	for {
		_, b, err := conn.Read(ctx)
		if err != nil {
			log.Fatal(err)
		}
		var msg struct {
			ID     int
			Result struct{ Data string }
			Error  *struct{ Message string }
		}
		if json.Unmarshal(b, &msg) != nil || msg.ID != 1 {
			continue
		}
		if msg.Error != nil {
			log.Fatal(msg.Error.Message)
		}
		png, err := base64.StdEncoding.DecodeString(msg.Result.Data)
		if err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile(*out, png, 0o644); err != nil {
			log.Fatal(err)
		}
		return
	}
}
