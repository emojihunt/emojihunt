package main

import (
	"log"

	"github.com/getsentry/sentry-go"
)

type SentryConfig struct {
	DSN      string `json:"dsn"`
	IssueURL string `json:"issue_url"`
}

func InitializeSentry(prod bool, config *SentryConfig) error {
	var environment = "dev"
	if prod {
		environment = "prod"
	}
	var opts = sentry.ClientOptions{
		Dsn: config.DSN,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			if hint.OriginalException != nil {
				log.Printf("error: %s", hint.OriginalException)
			} else {
				log.Printf("error: %s", hint.RecoveredException)
			}
			for _, exception := range event.Exception {
				frames := exception.Stacktrace.Frames
				for i := len(frames) - 1; i >= 0; i-- {
					log.Printf("\t%s:%d", frames[i].AbsPath, frames[i].Lineno)
				}
			}
			return event
		},
		Environment: environment,
	}
	return sentry.Init(opts)
}
