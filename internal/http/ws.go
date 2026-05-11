// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"log/slog"
	"net/http"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// handleWebSocket handles GET /api/p/:project/ws
// Upgrades to WebSocket and forwards hub events to the client until disconnected.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		http.Error(w, "no project in context", http.StatusInternalServerError)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: s.allowedWSOrigins(),
	})
	if err != nil {
		slog.Error("ws: accept failed", "err", err)
		return
	}
	defer conn.CloseNow()

	// Auth check after upgrade. The requireAuth middleware exempts /ws so we
	// can return a WebSocket close code (4401) instead of HTTP 401. The JS
	// client treats 4401 as a terminal auth failure and stops reconnecting.
	if userFromCtx(r.Context()) == nil {
		conn.Close(4401, "unauthorized")
		return
	}

	ctx := r.Context()
	ch := make(chan []byte, 32)
	p.Hub.Register(ch)
	defer p.Hub.Unregister(ch)

	// Read loop: handle client messages (lock.heartbeat, subscribe) without blocking sends.
	go func() {
		for {
			var msg map[string]any
			if err := wsjson.Read(ctx, conn, &msg); err != nil {
				return
			}
			// lock.heartbeat is a no-op in M3 (lock manager arrives in M5).
		}
	}()

	// Write loop: deliver hub events to the client.
	for {
		select {
		case <-ctx.Done():
			conn.Close(websocket.StatusNormalClosure, "")
			return
		case data, ok := <-ch:
			if !ok {
				conn.Close(websocket.StatusNormalClosure, "")
				return
			}
			if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
				return
			}
		}
	}
}
