package discovery

import (
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
			(*cancel)()
		}
	} else {
		if emoji != "" {
			ctx, cancel := context.WithCancel(context.Background())
			go func(ctx context.Context) {
				log.Printf("emoji added! kicking off round creation for %q (with delay)", name)
				select {
				case <-ctx.Done():
					log.Printf("all emoji removed, cancelling round creation for %q", name)
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

				err := d.createRound(context.Background(), name)
				if err != nil {
					log.Printf("error creating round %q: %s", name, spew.Sdump(err))
					return
				}

				// remove round from persistent state (all done)
				d.state.Lock()
				delete(d.state.DiscoveryNewRounds, name)
				d.state.CommitAndUnlock()
			}(ctx)
			d.roundCreation[messageID] = &cancel
		}
	}
	return nil
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
