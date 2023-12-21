package discovery

import (
	"log"
	"time"

	"github.com/davecgh/go-spew/spew"
	"golang.org/x/net/context"
	"golang.org/x/xerrors"
)

func (p *Poller) IsRoundNotification(messageID string) string {
	p.state.Lock()
	defer p.state.Unlock()

	for name, round := range p.state.DiscoveryNewRounds {
		if round.MessageID == messageID {
			return name
		}
	}
	return ""
}

func (p *Poller) StartOrCancelRoundCreation(name, messageID string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	emoji, err := p.getTopReaction(messageID)
	if err != nil {
		return err
	}

	if cancel, ok := p.roundCreation[messageID]; ok {
		if emoji == "" {
			delete(p.roundCreation, messageID)
			(*cancel)()
		}
	} else {
		if emoji != "" {
			ctx, cancel := context.WithCancel(p.main)
			// panics bubble up to the poller or discord handler
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
				p.mutex.Lock()
				delete(p.roundCreation, messageID)
				cancel() // must call to avoid goroutine leak
				p.mutex.Unlock()

				// remove round from persistent state (all done)
				p.state.Lock()
				roundInfo, ok := p.state.DiscoveryNewRounds[name]
				delete(p.state.DiscoveryNewRounds, name)
				p.state.CommitAndUnlock(ctx)

				var err error
				if !ok {
					err = xerrors.Errorf("round not found in state")
				} else {
					err = p.createRound(ctx, name, roundInfo)
				}
				if err != nil {
					log.Printf("error creating round %q: %s", name, spew.Sdump(err))
					return
				}
			}(ctx)
			p.roundCreation[messageID] = &cancel
		}
	}
	return nil
}

func (p *Poller) InitializeRoundCreation(ctx context.Context) {
	p.state.Lock()
	defer p.state.CommitAndUnlock(ctx)

	for name, round := range p.state.DiscoveryNewRounds {
		err := p.StartOrCancelRoundCreation(name, round.MessageID)
		if err != nil {
			// new-round notification has probably been deleted
			log.Printf("error kicking off round creation for %q, resetting round (%s)",
				name, spew.Sdump(err))
			delete(p.state.DiscoveryNewRounds, name)
		}
	}
}

func (p *Poller) CancelAllRoundCreation() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for messageID, cancel := range p.roundCreation {
		delete(p.roundCreation, messageID)
		(*cancel)()
	}
}

func (p *Poller) getTopReaction(messageID string) (string, error) {
	msg, err := p.discord.GetMessage(p.discord.QMChannel, messageID)
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
