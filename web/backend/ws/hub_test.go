package ws

import (
	"encoding/json"
	"net/http"
	"sync"
	"testing"
)

func TestNewHub(t *testing.T) {
	h := NewHub()
	if h == nil {
		t.Fatal("NewHub returned nil")
	}
	if h.clients == nil {
		t.Error("clients map not initialized")
	}
	if h.broadcast == nil {
		t.Error("broadcast channel not initialized")
	}
	if h.register == nil {
		t.Error("register channel not initialized")
	}
	if h.unregister == nil {
		t.Error("unregister channel not initialized")
	}
}

func TestClientCount_Empty(t *testing.T) {
	h := NewHub()
	if h.ClientCount() != 0 {
		t.Errorf("expected 0 clients, got %d", h.ClientCount())
	}
}

func TestClientCount_WithClients(t *testing.T) {
	h := NewHub()
	go h.Run()

	c1 := &Client{hub: h, send: make(chan []byte, 256), UserID: "u1"}
	c2 := &Client{hub: h, send: make(chan []byte, 256), UserID: "u2"}

	h.register <- c1
	h.register <- c2

	// Wait for registration to process
	// We use unregister+register to ensure previous ops are done
	done := make(chan struct{})
	go func() {
		h.register <- &Client{hub: h, send: make(chan []byte, 256), UserID: "sync"}
		close(done)
	}()
	<-done

	count := h.ClientCount()
	if count != 3 {
		t.Errorf("expected 3 clients, got %d", count)
	}
}

func TestCloseSend_Idempotent(t *testing.T) {
	c := &Client{send: make(chan []byte, 1)}

	// First close should work
	c.closeSend()

	// Second close should not panic
	c.closeSend()

	// Third close should not panic either
	c.closeSend()
}

func TestCloseSend_Concurrent(t *testing.T) {
	c := &Client{send: make(chan []byte, 1)}
	var wg sync.WaitGroup

	// Multiple goroutines closing simultaneously should not panic
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.closeSend()
		}()
	}
	wg.Wait()
}

func TestHubRegisterUnregister(t *testing.T) {
	h := NewHub()
	go h.Run()

	c1 := &Client{hub: h, send: make(chan []byte, 256), UserID: "u1"}
	c2 := &Client{hub: h, send: make(chan []byte, 256), UserID: "u2"}

	h.register <- c1
	h.register <- c2

	// Use a broadcast to sync — after broadcast is processed, registrations are done
	h.Broadcast(Message{Type: "sync"})
	<-c1.send
	<-c2.send

	if h.ClientCount() != 2 {
		t.Errorf("expected 2 clients after register, got %d", h.ClientCount())
	}

	h.unregister <- c1

	// Sync again: register a new client, then broadcast to confirm unregister processed
	c3 := &Client{hub: h, send: make(chan []byte, 256), UserID: "u3"}
	h.register <- c3
	h.Broadcast(Message{Type: "sync"})
	<-c2.send
	<-c3.send

	if h.ClientCount() != 2 {
		t.Errorf("expected 2 clients after unregister, got %d", h.ClientCount())
	}
}

func TestHubBroadcast(t *testing.T) {
	h := NewHub()
	go h.Run()

	c1 := &Client{hub: h, send: make(chan []byte, 256), UserID: "u1"}
	c2 := &Client{hub: h, send: make(chan []byte, 256), UserID: "u2"}

	h.register <- c1
	h.register <- c2

	// Sync
	sync := &Client{hub: h, send: make(chan []byte, 256), UserID: "sync"}
	h.register <- sync

	msg := Message{Type: "test_event", Data: map[string]string{"key": "value"}}
	h.Broadcast(msg)

	// Read from client channels
	got1 := <-c1.send
	got2 := <-c2.send

	var m1 Message
	if err := json.Unmarshal(got1, &m1); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}
	if m1.Type != "test_event" {
		t.Errorf("message type = %q, want %q", m1.Type, "test_event")
	}

	var m2 Message
	if err := json.Unmarshal(got2, &m2); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}
	if m2.Type != "test_event" {
		t.Errorf("message type = %q, want %q", m2.Type, "test_event")
	}
}

func TestMessageJSON(t *testing.T) {
	msg := Message{Type: "analysis_update", Data: map[string]int{"progress": 50}}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if decoded.Type != "analysis_update" {
		t.Errorf("Type = %q, want %q", decoded.Type, "analysis_update")
	}
}

func TestCheckOrigin_Localhost(t *testing.T) {
	t.Setenv("ENV", "")
	t.Setenv("ALLOWED_ORIGIN", "")

	origins := []struct {
		origin string
		want   bool
	}{
		{"", true},
		{"http://localhost:3000", true},
		{"http://localhost:5173", true},
		{"http://127.0.0.1:3000", true},
		{"https://myapp.fly.dev", true},
		{"https://evil.example.com", false},
	}

	for _, tt := range origins {
		r := &http.Request{Header: http.Header{}}
		if tt.origin != "" {
			r.Header.Set("Origin", tt.origin)
		}
		got := checkOrigin(r)
		if got != tt.want {
			t.Errorf("checkOrigin(%q) = %v, want %v", tt.origin, got, tt.want)
		}
	}
}

func TestCheckOrigin_AllowedOrigin(t *testing.T) {
	t.Setenv("ENV", "")
	t.Setenv("ALLOWED_ORIGIN", "https://myapp.example.com")

	r := &http.Request{Header: http.Header{}}
	r.Header.Set("Origin", "https://myapp.example.com")
	if !checkOrigin(r) {
		t.Error("expected allowed origin to pass")
	}

	r2 := &http.Request{Header: http.Header{}}
	r2.Header.Set("Origin", "https://other.example.com")
	if checkOrigin(r2) {
		t.Error("expected non-allowed origin to fail")
	}
}

func TestCheckOrigin_Development(t *testing.T) {
	t.Setenv("ENV", "development")
	t.Setenv("ALLOWED_ORIGIN", "")

	r := &http.Request{Header: http.Header{}}
	r.Header.Set("Origin", "https://anything.example.com")
	if !checkOrigin(r) {
		t.Error("expected all origins allowed in development")
	}
}
