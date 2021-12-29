package server

import (
	"crypto/subtle"
	"fmt"
	"net/http"

	"github.com/gauravjsingh/emojihunt/client"
)

type Server struct {
	air    *client.Airtable
	dis    *client.Discord
	drive  *client.Drive
	secret string
}

func New(air *client.Airtable, dis *client.Discord, drive *client.Drive, secret string) Server {
	return Server{air, dis, drive, secret}
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
	if subtle.ConstantTimeCompare(
		[]byte(s.secret),
		[]byte(r.URL.Query().Get("token"))) == 0 {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("incorrect token"))
		return
	}

	id := r.URL.Query().Get("record")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "error: record is required\n")
		return
	}

	record, err := s.air.FindByID(id)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "error: %#v\n", err)
		return
	}

	fmt.Fprintf(w, "Hi there! %#v\n", record)
}
