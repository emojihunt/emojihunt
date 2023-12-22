package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"time"

	_ "net/http/pprof"

	"github.com/emojihunt/emojihunt/bot"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/drive"
	"github.com/emojihunt/emojihunt/server"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/sync"
	"github.com/getsentry/sentry-go"
)

var prod = flag.Bool("prod", false, "selects development or production")

func init() { flag.Parse() }

func main() {
	// Initialize Sentry
	dsn, ok := os.LookupEnv("SENTRY_DSN")
	if !ok {
		panic("SENTRY_DSN is required")
	}
	sentry.Init(sentry.ClientOptions{
		Dsn: dsn,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			if hint.OriginalException != nil {
				log.Printf("error: %s", hint.OriginalException)
			} else {
				log.Printf("error: %s", hint.RecoveredException)
			}
			for _, exception := range event.Exception {
				if tr := exception.Stacktrace; tr != nil {
					for i := len(tr.Frames) - 1; i >= 0; i-- {
						log.Printf("\t%s:%d", tr.Frames[i].AbsPath, tr.Frames[i].Lineno)
					}
				}
			}
			return event
		},
	})
	defer sentry.Flush(time.Second * 5)
	defer func() {
		if err := recover(); err != nil {
			sentry.CurrentHub().Recover(err)
			panic(err)
		}
	}()

	// Set up the main context, which is cancelled on Ctrl-C
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() { <-ch; cancel() }()

	// Open database connection
	var state = state.New(ctx, "db.sqlite")

	// Set up clients
	var discord = discord.Connect(ctx, *prod, state)
	defer discord.Close()
	var drive = drive.NewClient(ctx, *prod)

	// Start internal engines
	log.Printf("starting syncer")
	var sync = sync.New(discord, drive, state, false)
	go sync.RestorePlaceholderEvent()
	sync.UpdateBotStatus(ctx)

	log.Printf("starting web server")
	server.Start(ctx, *prod, discord, state)

	log.Printf("starting discord bots")
	discord.RegisterBots(
		bot.NewEmojiNameBot(),
		bot.NewHuntYetBot(),
		bot.NewPuzzleBot(discord, state, sync),
		bot.NewQMBot(discord, state, sync),
		bot.NewReminderBot(ctx, discord, state),
	)

	log.Print("press ctrl+C to exit")
	<-ctx.Done()
}
