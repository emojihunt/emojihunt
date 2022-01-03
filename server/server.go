package server

import (
	"crypto/subtle"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/gauravjsingh/emojihunt/client"
	"github.com/gauravjsingh/emojihunt/schema"
	"github.com/gauravjsingh/emojihunt/syncer"
)

type Server struct {
	airtable *client.Airtable
	syncer   *syncer.Syncer
	secret   string
	origin   string
}

func New(airtable *client.Airtable, syncer *syncer.Syncer, secret, origin string) Server {
	return Server{airtable, syncer, secret, origin}
}

func (s *Server) ResyncURL(puzzle *schema.Puzzle) string {
	return fmt.Sprintf(
		"%s/resync?token=%s&record=%s",
		s.origin, s.secret, puzzle.AirtableRecord.ID,
	)
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

	log.Printf("Received HTTP request: %s", r.URL.Path)

	if strings.Contains(r.Header.Get("User-Agent"), "Discordbot") {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("ignoring request with discordbot user agent"))
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

	_, err = s.syncer.ForceUpdate(r.Context(), puzzle)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		spew.Fdump(w, err)
		return
	}

	fmt.Fprintf(w, "Update succeeded! %#v\n", puzzle)
}
