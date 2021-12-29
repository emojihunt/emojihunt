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
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/resync", s.resync)
		err := http.ListenAndServeTLS(":443", certFile, keyFile, mux)
		panic(err)
	}()
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			target := *r.URL
			target.Host = r.Host
			target.Scheme = "https"
			http.Redirect(w, r, target.String(), http.StatusTemporaryRedirect)
		})
		err := http.ListenAndServe(":80", mux)
		panic(err)
	}()
}

func (s *Server) resync(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hi there!")
}
