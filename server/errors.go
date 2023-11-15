package server

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/emojihunt/emojihunt/db"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v4"
	"github.com/mattn/go-sqlite3"
)

type APIErrorType string

const (
	ErrNotFound        APIErrorType = "not_found"
	ErrUnknownKey      APIErrorType = "unknown_key"
	ErrInvalidValue    APIErrorType = "invalid_value"
	ErrDuplicateRecord APIErrorType = "duplicate_record"
	ErrServerError     APIErrorType = "server_error"
)

type APIError struct {
	Type      APIErrorType `json:"type"`
	Message   string       `json:"message,omitempty"`
	Field     string       `json:"field,omitempty"`
	SentryURL string       `json:"sentry_url,omitempty"`
}

func (e APIError) Error() string {
	var s = fmt.Sprintf("api error: %s", e.Type)
	if e.Field != "" {
		s = fmt.Sprintf("%s in %q", s, e.Field)
	}
	return s
}

func (e APIError) StatusCode() int {
	switch e.Type {
	case ErrNotFound:
		return http.StatusNotFound
	case ErrUnknownKey:
		return http.StatusBadRequest
	case ErrInvalidValue:
		return http.StatusBadRequest
	case ErrDuplicateRecord:
		return http.StatusBadRequest
	case ErrServerError:
		return http.StatusInternalServerError
	default:
		panic("unknown error type: " + e.Type)
	}
}

func (s *Server) ErrorHandler(err error, c echo.Context) {
	var code int
	var response APIError

	if ai, ok := err.(APIError); ok {
		code = ai.StatusCode()
		response = ai
	} else {
		code = http.StatusInternalServerError
		response = APIError{
			Type:    ErrServerError,
			Message: err.Error(),
		}

		// Report unexpected errors to Sentry
		hub, ok := c.Get(sentryContextKey).(*sentry.Hub)
		if !ok {
			hub = sentry.CurrentHub().Clone()
		}
		event := hub.CaptureException(err)
		if event != nil {
			response.SentryURL = fmt.Sprintf(s.sentryURL, *event)
		}
	}

	// See https://github.com/labstack/echo/blob/master/echo.go
	if c.Response().Committed {
		return
	}
	if c.Request().Method == http.MethodHead {
		err = c.NoContent(code)
	} else {
		err = c.JSON(code, response)
	}
	if err != nil {
		log.Printf("error replying with error: %v", err)
	}
}

func translateError(err error) error {
	var ve db.ValidationError
	if ok := errors.As(err, &ve); ok {
		return APIError{
			Type: ErrInvalidValue, Message: err.Error(), Field: ve.Field,
		}
	}
	var se sqlite3.Error
	ok := errors.As(err, &se)
	if ok && se.ExtendedCode == sqlite3.ErrConstraintUnique {
		var parts = strings.Split(err.Error(), ".")
		return APIError{Type: ErrDuplicateRecord, Field: parts[len(parts)-1]}
	}
	if errors.Is(err, sql.ErrNoRows) {
		return APIError{Type: ErrNotFound}
	}
	return err // pass through
}
