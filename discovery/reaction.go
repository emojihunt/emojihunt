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

	for name, roundMesageID := range d.state.DiscoveryNewRounds {
		if roundMesageID == messageID {
			return name
		}
	}
	return ""
}

func (d *Poller) startOrCancelRoundCreation(name, messageID string) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	msg, err := d.discord.GetMessage(d.discord.QMChannel, messageID)
	if err != nil {
		return err
	}

	if cancel, ok := d.roundCreation[messageID]; ok {
		if len(msg.Reactions) == 0 {
			(*cancel)()
		}
	} else {
		if len(msg.Reactions) > 0 {
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
				defer cancel()

				err := d.createRound(name, messageID)
				if err != nil {
					log.Printf("error creating round %q: %s", name, spew.Sdump(err))
					return
				}

				d.state.Lock()
				defer d.state.CommitAndUnlock()
				delete(d.state.DiscoveryNewRounds, name)
			}(ctx)
			d.roundCreation[messageID] = &cancel
		}
	}
	return nil
}

func (d *Poller) createRound(name, messageID string) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	delete(d.roundCreation, messageID)

	// TODO
	return nil
}
