package server

import (
    "log"
    "net/http"
    "strings"
    "sync"
    "time"

    "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    HandshakeTimeout: 5 * time.Second,
    CheckOrigin:      func(r *http.Request) bool { return true },
}

// Hub routes judge verdicts to waiting WebSocket clients.
type Hub struct {
    mu   sync.Mutex
    subs map[int][]chan string
}

func NewHub() *Hub {
    return &Hub{subs: make(map[int][]chan string)}
}

// Subscribe registers a buffered channel for the given submission ID.
func (h *Hub) Subscribe(submissionID int) chan string {
    ch := make(chan string, 1)
    h.mu.Lock()
    h.subs[submissionID] = append(h.subs[submissionID], ch)
    h.mu.Unlock()
    return ch
}

// Unsubscribe removes a channel, cleaning up the map entry when empty.
func (h *Hub) Unsubscribe(submissionID int, ch chan string) {
    h.mu.Lock()
    defer h.mu.Unlock()
    chans := h.subs[submissionID]
    for i, c := range chans {
        if c == ch {
            h.subs[submissionID] = append(chans[:i], chans[i+1:]...)
            break
        }
    }
    if len(h.subs[submissionID]) == 0 {
        delete(h.subs, submissionID)
    }
}

// Notify sends a verdict to all subscribers for the given submission, then removes them.
func (h *Hub) Notify(submissionID int, verdict string) {
    h.mu.Lock()
    chans := h.subs[submissionID]
    delete(h.subs, submissionID)
    h.mu.Unlock()

    for _, ch := range chans {
        select {
        case ch <- verdict:
        default:
        }
    }
}

// submissionLiveHandler upgrades to WebSocket and pushes the verdict when ready.
//
// Subscribe-then-check ordering avoids the race between judge finishing and WS connecting:
//  1. subscribe to hub
//  2. check current status in store
//  3. if already complete → send immediately; otherwise wait for hub notification
func (s *Server) submissionLiveHandler(w http.ResponseWriter, r *http.Request) {
    // path: /submissions/:id/live
    trimmed := strings.TrimSuffix(r.URL.Path, "/live")
    id, err := parseID(trimmed, "/submissions/")
    if err != nil {
        http.NotFound(w, r)
        return
    }

    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("ws upgrade error for submission %d: %v", id, err)
        return
    }
    defer conn.Close()

    // 1. Subscribe before checking store to avoid missing the notification.
    ch := s.hub.Subscribe(id)
    defer s.hub.Unsubscribe(id, ch)

    // 2. Check if judge already finished.
    status, err := s.store.GetSubmissionStatus(r.Context(), id)
    if err == nil && status.Status == "complete" {
        sendWSVerdict(conn, status.Verdict)
        return
    }

    // 3. Wait for hub notification (60 s timeout).
    select {
    case verdict := <-ch:
        sendWSVerdict(conn, verdict)
    case <-time.After(60 * time.Second):
        sendWSVerdict(conn, "timeout")
    }
}

func sendWSVerdict(conn *websocket.Conn, verdict string) {
    type msg struct {
        Verdict string `json:"verdict"`
    }
    if err := conn.WriteJSON(msg{Verdict: verdict}); err != nil {
        log.Printf("ws write error: %v", err)
    }
}
