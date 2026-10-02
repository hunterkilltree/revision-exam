package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"quizlive/internal/store"
)

const good = `{"questions":[{"id":"q1","text":"T","options":[{"key":"A","text":"a"},{"key":"B","text":"b"}]}]}`

func upload(t *testing.T, url, body string) (*http.Response, map[string]any) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "data.json")
	fw.Write([]byte(body))
	mw.Close()
	resp, err := http.Post(url, mw.FormDataContentType(), &buf)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	json.NewDecoder(resp.Body).Decode(&out)
	return resp, out
}

func read(t *testing.T, ctx context.Context, c *websocket.Conn, typ string) map[string]any {
	for {
		_, b, err := c.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		json.Unmarshal(b, &m)
		if m["type"] == typ {
			return m
		}
	}
}

func TestEndToEnd(t *testing.T) {
	ts := httptest.NewServer(New(Config{PublicBaseURL: "http://app", AllowedOrigins: []string{"*"}}, store.NewMemory(5)))
	defer ts.Close()

	if resp, out := upload(t, ts.URL+"/sessions", `{"questions":[{"text":"x"}]}`); resp.StatusCode != 422 || out["errors"] == nil {
		t.Fatalf("expected 422 with errors, got %d %v", resp.StatusCode, out)
	}
	resp, out := upload(t, ts.URL+"/sessions", good)
	if resp.StatusCode != 201 {
		t.Fatalf("create: %d %v", resp.StatusCode, out)
	}
	id, key := out["sessionId"].(string), out["hostKey"].(string)
	if !strings.HasSuffix(out["hostLink"].(string), "/admin/"+id+"?key="+key) || out["shareLink"] != "http://app/join/"+id {
		t.Fatalf("links: %v", out)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/" + id

	if _, r, err := websocket.Dial(ctx, wsURL+"?key=wrong", nil); err == nil || r == nil || r.StatusCode != 403 {
		t.Fatal("wrong key must be rejected with 403")
	}
	admin, _, err := websocket.Dial(ctx, wsURL+"?key="+key, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.CloseNow()
	client, _, _ := websocket.Dial(ctx, wsURL, nil)
	defer client.CloseNow()

	client.Write(ctx, websocket.MessageText, []byte(`{"type":"join","name":"Ann"}`))
	read(t, ctx, client, "joined")
	client.Write(ctx, websocket.MessageText, []byte(`garbage`)) // must not crash anything
	admin.Write(ctx, websocket.MessageText, []byte(`{"type":"start","durationSec":10}`))
	read(t, ctx, client, "state")
	client.Write(ctx, websocket.MessageText, []byte(`{"type":"answer","questionId":"q1","key":"B"}`))
	admin.Write(ctx, websocket.MessageText, []byte(`{"type":"finish"}`))
	for {
		st := read(t, ctx, admin, "state")
		if st["phase"] == "results" {
			if st["tally"].(map[string]any)["B"].(float64) != 1 {
				t.Fatalf("tally: %v", st["tally"])
			}
			break
		}
	}
}
