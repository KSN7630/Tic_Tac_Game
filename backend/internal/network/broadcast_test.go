package network

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestConcurrentBroadcastsNoRace verifies that many goroutines broadcasting to
// the same set of connections never write concurrently to a single websocket
// connection. Run with `go test -race` to catch the regression where two
// goroutines wrote the same conn and the client got disconnected.
func TestConcurrentBroadcastsNoRace(t *testing.T) {
	nb := NewNetBroadcaster()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		// Same connection registered under multiple rooms, like a real player
		// who is in the lobby and a game room at once.
		nb.RegisterConn("lobby", conn)
		nb.RegisterConn("room1", conn)
		// Drain incoming frames until the client closes.
		go func() {
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}()
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	const clients = 3
	conns := make([]*websocket.Conn, 0, clients)
	for i := 0; i < clients; i++ {
		c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("dial: %v", err)
		}
		conns = append(conns, c)
		// Client-side reader so the server writes don't block.
		go func(c *websocket.Conn) {
			for {
				if _, _, err := c.ReadMessage(); err != nil {
					return
				}
			}
		}(c)
	}
	defer func() {
		for _, c := range conns {
			_ = c.Close()
		}
	}()

	// Give the server time to register all connections.
	time.Sleep(100 * time.Millisecond)

	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = nb.BroadcastToRoom(ctx, "room1", "move_made", map[string]any{"n": 1})
		}()
		go func() {
			defer wg.Done()
			_ = nb.BroadcastToAll(map[string]any{"msg": "hi"})
		}()
	}
	wg.Wait()
}

// TestBroadcastToAllDeduplicates ensures a connection registered under several
// rooms receives a global broadcast exactly once.
func TestBroadcastToAllDeduplicates(t *testing.T) {
	nb := NewNetBroadcaster()

	received := make(chan struct{}, 16)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		nb.RegisterConn("lobby", conn)
		nb.RegisterConn("room1", conn)
		nb.RegisterConn("room2", conn)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	go func() {
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
			received <- struct{}{}
		}
	}()

	time.Sleep(100 * time.Millisecond)
	if err := nb.BroadcastToAll(map[string]any{"msg": "once"}); err != nil {
		t.Fatalf("broadcast: %v", err)
	}

	// Expect exactly one delivery even though the conn is in 3 rooms.
	got := 0
	timeout := time.After(500 * time.Millisecond)
loop:
	for {
		select {
		case <-received:
			got++
		case <-timeout:
			break loop
		}
	}
	if got != 1 {
		t.Fatalf("expected exactly 1 delivery, got %d", got)
	}
}
