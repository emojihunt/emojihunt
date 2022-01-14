package server

import (
	"crypto/subtle"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/emojihunt/emojihunt/client"
	"github.com/emojihunt/emojihunt/schema"
	"github.com/emojihunt/emojihunt/syncer"
)

type ServerConfig struct {
	SecretToken     string `json:"secret_token"`
	CertificateFile string `json:"certificate_file"`
	KeyFile         string `json:"key_file"`
	Origin          string `json:"origin"`
}

type Server struct {
	airtable       *client.Airtable
	syncer         *syncer.Syncer
	secret, origin string
}

func Start(airtable *client.Airtable, syncer *syncer.Syncer, config *ServerConfig) error {
	if config.SecretToken == "" {
		return fmt.Errorf("secret token cannot be empty")
	}
	server := &Server{airtable, syncer, config.SecretToken, config.Origin}
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/resync", server.resync)
		err := http.ListenAndServeTLS(":443", config.CertificateFile, config.KeyFile, mux)
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
	return nil
}

func (s *Server) ResyncURL(puzzle *schema.Puzzle) string {
	return fmt.Sprintf(
		"%s/resync?token=%s&record=%s",
		s.origin, s.secret, puzzle.AirtableRecord.ID,
	)
}

func (s *Server) resync(w http.ResponseWriter, r *http.Request) {
	if subtle.ConstantTimeCompare(
		[]byte(s.secret),
		[]byte(r.URL.Query().Get("token"))) == 0 {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("incorrect token"))
		return
	}

	if strings.Contains(r.Header.Get("User-Agent"), "Discordbot") {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("ignoring request with discordbot user agent"))
		log.Printf("ignoring HTTP request: %q %q", r.URL.Path, r.Header.Get("User-Agent"))
		return
	}

	log.Printf("processing re-sync request: %q", r.URL.Query().Get("record"))

	id := r.URL.Query().Get("record")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "error: record is required\n")
		return
	}

	puzzle, err := s.airtable.LockByID(id)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "error: %#v\n", err)
		return
	}
	defer puzzle.Unlock()

	_, err = s.syncer.ForceUpdate(r.Context(), puzzle)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		spew.Fdump(w, err)
		return
	}

	fmt.Fprintf(w, "Update succeeded! %#v\n", puzzle)
}
