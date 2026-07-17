package webserver

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"uit-toolbox/config"

	"golang.org/x/net/websocket"
)

// Echo the data received on the WebSocket.
func EchoServer(ws *websocket.Conn) {
	io.Copy(ws, ws)
}

// This example demonstrates a trivial echo server.
func StartWebSocket(ctx context.Context) error {
	log := config.GetLogger().With(slog.String("func", "StartWebSocket"))
	http.Handle("/ws", websocket.Handler(EchoServer))
	log.Info("Starting WebSocket server...")
	err := http.ListenAndServe(":1411", nil)
	if err != nil {
		return err
	}
	return nil
}
