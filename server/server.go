package server

import (
	"fmt"
	"net/http"

	"github.com/gauravjsingh/emojihunt/client"
)

type Server struct {
	air   *client.Airtable
	dis   *client.Discord
	drive *client.Drive
}

func New(air *client.Airtable, dis *client.Discord, drive *client.Drive) Server {
	return Server{air, dis, drive}
}

func (s *Server) Start(certFile, keyFile string) {
	http.HandleFunc("/resync", s.resync)
	http.ListenAndServeTLS(":443", certFile, keyFile, nil)
}

func (s *Server) resync(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintln(w, "Hi there!")
}
