package server

import (
	"testing"
	"time"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
)

func TestKeepAliveState(t *testing.T) {
	const (
		timeout  = 30 * time.Second
		interval = 15 * time.Second
	)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	srv := &Server{KeepAliveTimeout: timeout, KeepAliveInterval: interval}

	conn := func(state mc.State, pending bool) *Connection {
		return &Connection{
			Server:            srv,
			State:             state,
			KeepAlivePending:  pending,
			LastKeepAliveSent: base,
		}
	}

	tests := []struct {
		name        string
		state       mc.State
		pending     bool
		elapsed     time.Duration
		wantSend    bool
		wantTimeout bool
	}{
		{"idle, before interval", mc.StatePlay, false, interval - time.Second, false, false},
		{"idle, interval elapsed sends a challenge", mc.StatePlay, false, interval, true, false},
		{"pending, within timeout waits", mc.StatePlay, true, timeout - time.Second, false, false},
		{"pending, no second challenge while one is in flight", mc.StatePlay, true, interval + time.Second, false, false},
		{"pending, timeout exceeded disconnects", mc.StatePlay, true, timeout + time.Second, false, true},
		{"configuration state is checked too", mc.StateConfiguration, false, interval, true, false},
		{"login state is ignored", mc.StateLogin, true, timeout + time.Hour, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			send, timedOut := conn(tt.state, tt.pending).keepAliveState(base.Add(tt.elapsed))
			if send != tt.wantSend || timedOut != tt.wantTimeout {
				t.Errorf("keepAliveState(elapsed=%s) = (send=%v, timeout=%v); want (send=%v, timeout=%v)",
					tt.elapsed, send, timedOut, tt.wantSend, tt.wantTimeout)
			}
		})
	}
}

func TestAcceptKeepAlive(t *testing.T) {
	const challenge = int64(1_700_000_000_000)

	t.Run("matching id while pending is accepted", func(t *testing.T) {
		c := &Connection{KeepAlivePending: true, LastKeepAliveID: challenge}
		if !c.acceptKeepAlive(challenge) {
			t.Fatal("valid keep-alive response was rejected")
		}
		if c.KeepAlivePending {
			t.Error("KeepAlivePending must clear after a valid response")
		}
	})

	t.Run("wrong id is rejected", func(t *testing.T) {
		c := &Connection{KeepAlivePending: true, LastKeepAliveID: challenge}
		if c.acceptKeepAlive(challenge + 1) {
			t.Fatal("keep-alive with a mismatched id was accepted")
		}
		if !c.KeepAlivePending {
			t.Error("KeepAlivePending must stay set after a rejected response")
		}
	})

	t.Run("unsolicited response is rejected", func(t *testing.T) {
		c := &Connection{KeepAlivePending: false, LastKeepAliveID: challenge}
		if c.acceptKeepAlive(challenge) {
			t.Fatal("keep-alive was accepted with no challenge pending")
		}
	})
}
