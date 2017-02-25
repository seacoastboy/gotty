package server

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	"github.com/yudai/gotty/pkg/homedir"
)

func (server *Server) handleMain(w http.ResponseWriter, r *http.Request) {
	server.stopTimer()
	connections := atomic.AddInt64(server.connections, 1)
	defer func() {
		connections := atomic.AddInt64(server.connections, -1)

		log.Printf(
			"Connection closed: %s, connections: %d/%d",
			r.RemoteAddr, connections, server.options.MaxConnection,
		)
		if connections == 0 {
			server.resetTimer()
		}
	}()

	log.Printf("New client connected: %s", r.RemoteAddr)
	if int64(server.options.MaxConnection) != 0 {
		if connections >= int64(server.options.MaxConnection) {
			log.Printf("Reached max connection: %d", server.options.MaxConnection)
			return
		}
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	conn, err := server.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade connection: "+err.Error(), 500)
		return
	}
	defer conn.Close()

	err = server.processWSConn(conn)
	if err != nil {
		log.Printf(err.Error())
		return
	}

	if server.options.Once {
		//todo
	}
}

func (server *Server) processWSConn(conn *websocket.Conn) error {
	_, initLine, err := conn.ReadMessage()
	if err != nil {
		return errors.Wrapf(err, "failed to authenticate websocket connection")
	}

	var init InitMessage
	err = json.Unmarshal(initLine, &init)
	if err != nil {
		return errors.Wrapf(err, "failed to authenticate websocket connection")
	}
	if init.AuthToken != server.options.Credential {
		return errors.New("failed to authenticate websocket connection")
	}

	var queryPath string
	if server.options.PermitArguments && init.Arguments != "" {
		queryPath = init.Arguments
	} else {
		queryPath = "?"
	}

	query, err := url.Parse(queryPath)
	if err != nil {
		return errors.Wrapf(err, "failed to parse arguments")
	}
	params := query.Query()
	_, err = server.factory.New(params)
	if err != nil {
		return errors.Wrapf(err, "failed to create ")
	}

	return nil
}

func (server *Server) handleCustomIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, homedir.Expand(server.options.IndexFile))
}

func (server *Server) handleAuthToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Write([]byte("var gotty_auth_token = '" + server.options.Credential + "';"))
}
