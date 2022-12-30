package discovery

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
	"github.com/emojihunt/emojihunt/client"
	"golang.org/x/net/context"
)

func (d *Poller) RegisterReactionHandler(discord *client.Discord) {
	var handler client.DiscordReactionHandler = func(
		s *discordgo.Session, r *discordgo.MessageReaction, kind string) error {

		if d.state.IsKilled() {
			return nil
		}

		roundName := d.isRoundNotification(r.MessageID)
		if roundName == "" {
			return nil
		}

		log.Printf("discord: handling reaction %s%s on message %q from user %s",
			kind, r.Emoji.Name, r.MessageID, r.UserID)

		return d.startOrCancelRoundCreation(roundName, r.MessageID)
	}
	discord.AddReactionHandler(&handler)
}

func (d *Poller) isRoundNotification(messageID string) string {
	d.state.Lock()
	defer d.state.Unlock()

	for name, round := range d.state.DiscoveryNewRounds {
		if round.MessageID == messageID {
			return name
		}
	}
	return ""
}

func (d *Poller) startOrCancelRoundCreation(name, messageID string) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	emoji, err := d.getTopReaction(messageID)
	if err != nil {
		return err
	}

	if cancel, ok := d.roundCreation[messageID]; ok {
		if emoji == "" {
			delete(d.roundCreation, messageID)
			(*cancel)()
		}
	} else {
		if emoji != "" {
			ctx, cancel := context.WithCancel(context.Background())
			go func(ctx context.Context) {
				log.Printf("kicking off round creation for %q (with delay)", name)
				select {
				case <-ctx.Done():
					log.Printf("cancelling round creation for %q", name)
					return
				case <-time.After(roundCreationPause):
					break
				}

				log.Printf("creating round %q", name)

				// remove round from poller (can no longer be cancelled)
				d.mutex.Lock()
				delete(d.roundCreation, messageID)
				cancel() // must call to avoid goroutine leak
				d.mutex.Unlock()

				// remove round from persistent state (all done)
				d.state.Lock()
				roundInfo, ok := d.state.DiscoveryNewRounds[name]
				delete(d.state.DiscoveryNewRounds, name)
				d.state.CommitAndUnlock()

				var err error
				if !ok {
					err = fmt.Errorf("round not found in state")
				} else {
					err = d.createRound(context.Background(), name, roundInfo)
				}
				if err != nil {
					log.Printf("error creating round %q: %s", name, spew.Sdump(err))
					return
				}
			}(ctx)
			d.roundCreation[messageID] = &cancel
		}
	}
	return nil
}

func (d *Poller) InitializeRoundCreation() {
	d.state.Lock()
	defer d.state.CommitAndUnlock()

	for name, round := range d.state.DiscoveryNewRounds {
		err := d.startOrCancelRoundCreation(name, round.MessageID)
		if err != nil {
			// new-round notification has probably been deleted
			log.Printf("error kicking off round creation for %q, resetting round (%s)",
				name, spew.Sdump(err))
			delete(d.state.DiscoveryNewRounds, name)
		}
	}
}

func (d *Poller) CancelAllRoundCreation() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	for messageID, cancel := range d.roundCreation {
		delete(d.roundCreation, messageID)
		(*cancel)()
	}
}

func (d *Poller) getTopReaction(messageID string) (string, error) {
	msg, err := d.discord.GetMessage(d.discord.QMChannel, messageID)
	if err != nil {
		return "", err
	}

	emoji, count := "", 0
	for _, reaction := range msg.Reactions {
		if reaction.Count > count && reaction.Emoji.Name != "" {
			emoji = reaction.Emoji.Name
			count = reaction.Count
		}
	}
	return emoji, nil
}
