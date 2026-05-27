package server

import (
	"sync"
	"testing"
	"time"
)

func TestHubSubscribeReceivesVerdict(t *testing.T) {
	h := NewHub()
	ch := h.Subscribe(1)
	h.Notify(1, "accepted")

	select {
	case got := <-ch:
		if got != "accepted" {
			t.Errorf("want accepted, got %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for verdict")
	}
}

// TestHubBufferedChannelCapturesEarlyNotify verifies that Notify before the
// caller reads the channel still delivers the verdict (buffered channel design).
func TestHubBufferedChannelCapturesEarlyNotify(t *testing.T) {
	h := NewHub()
	ch := h.Subscribe(99)

	// Notify fires first — the buffered channel should hold it.
	h.Notify(99, "wrong_answer")

	// Read happens after Notify; must still succeed instantly.
	select {
	case got := <-ch:
		if got != "wrong_answer" {
			t.Errorf("want wrong_answer, got %q", got)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("verdict was not buffered — channel drained before read")
	}
}

func TestHubMultipleSubscribers(t *testing.T) {
	h := NewHub()
	ch1 := h.Subscribe(5)
	ch2 := h.Subscribe(5)

	h.Notify(5, "compile_error")

	for i, ch := range []chan string{ch1, ch2} {
		select {
		case got := <-ch:
			if got != "compile_error" {
				t.Errorf("subscriber %d: want compile_error, got %q", i, got)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d: timeout", i)
		}
	}
}

func TestHubUnsubscribeBeforeNotify(t *testing.T) {
	h := NewHub()
	ch := h.Subscribe(10)
	h.Unsubscribe(10, ch)
	h.Notify(10, "accepted")

	select {
	case got := <-ch:
		t.Errorf("unsubscribed channel should not receive, got %q", got)
	case <-time.After(50 * time.Millisecond):
		// correct — nothing received
	}
}

func TestHubNotifyNoSubscribersDoesNotPanic(t *testing.T) {
	h := NewHub()
	// Must not panic
	h.Notify(999, "accepted")
}

// TestHubCleansUpAfterNotify verifies the subs map entry is deleted after Notify.
func TestHubCleansUpAfterNotify(t *testing.T) {
	h := NewHub()
	h.Subscribe(7)
	h.Notify(7, "accepted")

	h.mu.Lock()
	_, exists := h.subs[7]
	h.mu.Unlock()

	if exists {
		t.Error("subs map should be empty after Notify, but entry still present")
	}
}

// TestHubUnsubscribeKeepsOtherSubscribers verifies that unsubscribing one channel
// still delivers the verdict to other subscribers on the same ID.
func TestHubUnsubscribeKeepsOtherSubscribers(t *testing.T) {
	h := NewHub()
	ch1 := h.Subscribe(20)
	ch2 := h.Subscribe(20)
	h.Unsubscribe(20, ch1)

	h.Notify(20, "accepted")

	select {
	case got := <-ch2:
		if got != "accepted" {
			t.Errorf("want accepted, got %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("ch2 timeout")
	}

	// ch1 should have nothing
	select {
	case got := <-ch1:
		t.Errorf("ch1 was unsubscribed but received %q", got)
	case <-time.After(20 * time.Millisecond):
	}
}

// TestHubConcurrentSubscribeAndNotify stress-tests the hub under concurrent
// access. Run with go test -race to catch data races.
func TestHubConcurrentSubscribeAndNotify(t *testing.T) {
	h := NewHub()
	const workers = 30

	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ch := h.Subscribe(id)
			h.Notify(id, "accepted")
			select {
			case <-ch:
			case <-time.After(2 * time.Second):
				t.Errorf("worker %d: timeout", id)
			}
		}(i)
	}
	wg.Wait()
}

// TestHubIndependentSubmissionIDs verifies that notifying one ID doesn't bleed
// into a subscriber waiting on a different ID.
func TestHubIndependentSubmissionIDs(t *testing.T) {
	h := NewHub()
	ch100 := h.Subscribe(100)
	h.Notify(200, "wrong_answer") // different ID

	select {
	case got := <-ch100:
		t.Errorf("ID 100 should not receive notification for ID 200, got %q", got)
	case <-time.After(50 * time.Millisecond):
		// correct
	}
}
