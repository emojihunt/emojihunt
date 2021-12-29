package server

import (
	"crypto/subtle"
	"fmt"
	"net/http"

	"github.com/gauravjsingh/emojihunt/client"
	"github.com/gauravjsingh/emojihunt/syncer"
)

type Server struct {
	airtable *client.Airtable
	syncer   *syncer.Syncer
	secret   string
}

func New(airtable *client.Airtable, syncer *syncer.Syncer, secret string) Server {
	return Server{airtable, syncer, secret}
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

	puzzle, err := s.airtable.FindByID(id)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "error: %#v\n", err)
		return
	}

	puzzle, err = s.syncer.IdempotentCreateUpdate(r.Context(), puzzle)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error: %#v\n", err)
		return
	}

	fmt.Fprintf(w, "Update succeeded! %#v\n", puzzle)
}
