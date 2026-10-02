package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"golang.org/x/time/rate"

	"quizlive/internal/session"
)

const (
	readLimit    = 4096
	writeTimeout = 10 * time.Second
	pingEvery    = 20 * time.Second
	sendBuffer   = 32
)

// wsSink adapts a websocket to session.Sink with a bounded queue so a slow peer never blocks the hub.
type wsSink struct {
	out    chan []byte
	cancel context.CancelFunc
}

func (s *wsSink) Send(b []byte) bool {
	select {
	case s.out <- b:
		return true
	default:
		return false
	}
}
func (s *wsSink) Close() { s.cancel() }

func (s *Server) ws(w http.ResponseWriter, r *http.Request) {
	hub, ok := s.store.Get(r.PathValue("id"))
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	admin := false
	if key := r.URL.Query().Get("key"); key != "" {
		if subtle.ConstantTimeCompare([]byte(key), []byte(hub.HostKey)) != 1 {
			http.Error(w, "invalid host key", http.StatusForbidden)
			return
		}
		admin = true
	}
	if !s.originAllowed(r.Header.Get("Origin")) {
		http.Error(w, "origin not allowed", http.StatusForbidden)
		return
	}
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true}) // origin checked above
	if err != nil {
		return
	}
	c.SetReadLimit(readLimit)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	sink := &wsSink{out: make(chan []byte, sendBuffer), cancel: cancel}
	conn := &session.Conn{Admin: admin, Sink: sink}
	hub.Attach(conn)
	defer hub.Detach(conn)

	go func() { // writer + keepalive
		t := time.NewTicker(pingEvery)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				c.Close(websocket.StatusGoingAway, "closing")
				return
			case b := <-sink.out:
				wctx, wc := context.WithTimeout(ctx, writeTimeout)
				err := c.Write(wctx, websocket.MessageText, b)
				wc()
				if err != nil {
					cancel()
					return
				}
			case <-t.C:
				pctx, pc := context.WithTimeout(ctx, writeTimeout)
				err := c.Ping(pctx)
				pc()
				if err != nil {
					cancel()
					return
				}
			}
		}
	}()

	limiter := rate.NewLimiter(10, 20)
	for {
		_, data, err := c.Read(ctx)
		if err != nil {
			return
		}
		if !limiter.Allow() {
			sink.Send([]byte(`{"type":"error","code":"rate_limited","message":"slow down"}`))
			continue
		}
		var m session.Msg
		if json.Unmarshal(data, &m) != nil { // malformed frames are ignored, never fatal
			sink.Send([]byte(`{"type":"error","code":"bad_request","message":"malformed message"}`))
			continue
		}
		hub.Handle(conn, m)
	}
}
