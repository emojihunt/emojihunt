package server

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/emojihunt/emojihunt/db"
	"github.com/emojihunt/emojihunt/schema"
	"github.com/emojihunt/emojihunt/syncer"
	"golang.org/x/xerrors"
)

type ServerConfig struct {
	SecretToken     string `json:"secret_token"`
	CertificateFile string `json:"certificate_file"`
	KeyFile         string `json:"key_file"`
	Origin          string `json:"origin"`
}

type Server struct {
	db             *db.Client
	syncer         *syncer.Syncer
	secret, origin string
}

func Start(db *db.Client, syncer *syncer.Syncer, config *ServerConfig) error {
	if config.SecretToken == "" {
		return xerrors.Errorf("secret token cannot be empty")
	}
	origin := config.Origin
	if origin == "" {
		origin = "http://localhost:8000"
	}
	server := &Server{db, syncer, config.SecretToken, origin}
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/robots.txt", server.robots)
		mux.HandleFunc("/resync", server.resync)
		err := http.ListenAndServe(":8000", mux)
		panic(err)
	}()

	return nil
}

func (s *Server) ResyncURL(puzzle *schema.Puzzle) string {
	return fmt.Sprintf(
		"%s/resync?token=%s&record=%d",
		s.origin, s.secret, puzzle.ID,
	)
}

func (s *Server) robots(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("User-agent: *\nDisallow: /\n"))
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

	id, err := strconv.ParseInt(r.URL.Query().Get("record"), 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "error: %#v\n", err)
		return
	}

	puzzle, err := s.db.LockByID(context.TODO(), id)
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
