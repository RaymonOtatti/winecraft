package server

import (
	"context"
	"crypto/subtle"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/coder/websocket"

	"github.com/RaymonOtatti/winecraft/internal/game"
	"github.com/RaymonOtatti/winecraft/internal/proto"
	"github.com/RaymonOtatti/winecraft/internal/rate"
)

const writeTimeout = 10 * time.Second

// client is one WebSocket connection. Only the hub sends on send, and only
// the hub closes it.
type client struct {
	id   uint32
	name string
	ip   string
	conn *websocket.Conn
	send chan []byte
}

// serveWS runs one connection: handshake, join, then a reader loop on this
// goroutine and a writer loop on another.
func (h *Hub) serveWS(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !h.gate.enter(ip, h.cfg.MaxPerIP) {
		http.Error(w, "too many connections", http.StatusTooManyRequests)
		return
	}
	defer h.gate.leave(ip)

	// Accept's default origin check only allows the page's own host.
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	conn.SetReadLimit(proto.MaxMessage) // bigger frames close with StatusMessageTooBig
	ctx := r.Context()
	c := &client{ip: ip, conn: conn, send: make(chan []byte, h.cfg.SendBuffer)}

	name, token, code, ok := h.handshake(ctx, c)
	if !ok {
		reject(ctx, conn, code)
		return
	}
	reply := make(chan error, 1)
	select {
	case h.joins <- joinReq{c: c, name: name, token: token, reply: reply}:
	case <-h.done:
		conn.Close(websocket.StatusGoingAway, "server shutting down")
		return
	}
	if err := <-reply; err != nil {
		code := proto.ErrServerFull
		if errors.Is(err, game.ErrBadName) {
			code = proto.ErrBadName
		}
		reject(ctx, conn, code)
		return
	}

	go c.writeLoop(ctx, h.cfg.Heartbeat)
	c.readLoop(ctx, h)
	select {
	case h.leaves <- c:
	case <-h.done:
	}
}

// handshake reads the first message, which must be a valid Hello.
func (h *Hub) handshake(ctx context.Context, c *client) (name, token string, errCode uint8, ok bool) {
	hctx, cancel := context.WithTimeout(ctx, h.cfg.HelloTimeout)
	defer cancel()
	typ, b, err := c.conn.Read(hctx)
	if err != nil || typ != websocket.MessageBinary {
		return "", "", 0, false
	}
	m, err := proto.Decode(b)
	if err != nil {
		return "", "", 0, false
	}
	hello, isHello := m.(*proto.Hello)
	switch {
	case !isHello:
		return "", "", 0, false
	case hello.Version != proto.Version:
		return "", "", proto.ErrBadVersion, false
	case h.gate.locked(c.ip, time.Now(), h.cfg.FailInterval, h.cfg.FailBurst):
		h.cfg.Log.Warn("join locked out after wrong codes", "ip", c.ip)
		return "", "", proto.ErrRateLimited, false
	case subtle.ConstantTimeCompare([]byte(hello.JoinCode), []byte(h.cfg.JoinCode)) != 1:
		h.gate.fail(c.ip, time.Now(), h.cfg.FailInterval, h.cfg.FailBurst)
		h.cfg.Log.Warn("bad join code", "ip", c.ip)
		return "", "", proto.ErrBadJoinCode, false
	}
	name, err = game.CleanName(hello.Name)
	if err != nil {
		return "", "", proto.ErrBadName, false
	}
	return name, hello.Token, 0, true
}

// reject tells the client why (when there is a code) and closes.
func reject(ctx context.Context, conn *websocket.Conn, code uint8) {
	if code != 0 {
		if b, err := proto.Encode(&proto.Error{Code: code}); err == nil {
			wctx, cancel := context.WithTimeout(ctx, writeTimeout)
			conn.Write(wctx, websocket.MessageBinary, b)
			cancel()
		}
	}
	conn.Close(websocket.StatusPolicyViolation, "rejected")
}

// readLoop forwards decoded messages to the hub. A client that sends
// nothing for cfg.Idle is dropped (the game client heartbeats well inside
// that), and one that floods past its message allowance is cut off.
func (c *client) readLoop(ctx context.Context, h *Hub) {
	var allowance rate.Bucket
	for {
		rctx, cancel := context.WithTimeout(ctx, h.cfg.Idle)
		typ, b, err := c.conn.Read(rctx)
		cancel()
		if err != nil {
			return
		}
		if !allowance.Take(time.Now(), h.cfg.MsgInterval, h.cfg.MsgBurst, 1) {
			h.cfg.Log.Warn("message flood", "id", c.id, "ip", c.ip)
			c.conn.Close(websocket.StatusPolicyViolation, "too many messages")
			return
		}
		if typ != websocket.MessageBinary {
			c.conn.Close(websocket.StatusUnsupportedData, "binary only")
			return
		}
		m, err := proto.Decode(b)
		if err != nil {
			c.conn.Close(websocket.StatusPolicyViolation, "malformed message")
			return
		}
		select {
		case h.inbound <- inbound{c: c, msg: m}:
		case <-h.done:
			return
		}
	}
}

// writeLoop drains the send queue and sends a game-level heartbeat, because
// the browser WebSocket cannot ping and Cloudflare drops idle connections.
func (c *client) writeLoop(ctx context.Context, heartbeat time.Duration) {
	ping := time.NewTicker(heartbeat)
	defer ping.Stop()
	var nonce uint32
	write := func(b []byte) bool {
		wctx, cancel := context.WithTimeout(ctx, writeTimeout)
		defer cancel()
		return c.conn.Write(wctx, websocket.MessageBinary, b) == nil
	}
	for {
		select {
		case b, ok := <-c.send:
			if !ok {
				c.conn.Close(websocket.StatusNormalClosure, "")
				return
			}
			if !write(b) {
				c.conn.CloseNow()
				return
			}
		case <-ping.C:
			nonce++
			if b, err := proto.Encode(&proto.Ping{Nonce: nonce}); err == nil && !write(b) {
				c.conn.CloseNow()
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// clientIP returns the real client address. CF-Connecting-IP is trusted only
// when the TCP peer is loopback, which is where cloudflared connects from;
// from anywhere else the header could be forged.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if peer := net.ParseIP(host); peer != nil && peer.IsLoopback() {
		if cf := net.ParseIP(r.Header.Get("CF-Connecting-IP")); cf != nil {
			return cf.String()
		}
	}
	return host
}
