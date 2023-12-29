package sync

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/state"
)

// CreateDiscordCategory creates a new Discord category and returns its ID.
func (c *Client) CreateDiscordCategory(ctx context.Context, round state.Round) (string, error) {
	log.Printf("sync: creating discord category for %q", round.Name)
	category, err := c.discord.CreateCategory(roundCategoryPrefix + round.Name)
	if err != nil {
		return "", err
	}
	return category.ID, nil
}

func (c *Client) RestoreSolvedCategories() error {
	categories, err := c.discord.GetChannelCategories()
	if err != nil {
		return err
	}
	var solved []string
	for i := 0; i < solvedCategoryCount; i++ {
		name := solvedCategoryPrefix + string(rune(int('A')+i))
		if category, ok := categories[name]; ok {
			solved = append(solved, category.ID)
		} else {
			log.Printf("sync: restoring category %q", name)
			category, err := c.discord.CreateCategory(name)
			if err != nil {
				return err
			}
			solved = append(solved, category.ID)
		}
	}
	c.solvedCategories = solved
	return nil
}

type DiscordCategoryFields struct {
	RoundName     string
	RoundCategory string
}

func NewDiscordCategoryFields(round state.Round) DiscordCategoryFields {
	return DiscordCategoryFields{
		RoundName:     round.Name,
		RoundCategory: round.DiscordCategory,
	}
}

// UpdateDiscordCategory configures the name of the round category.
func (c *Client) UpdateDiscordCategory(fields DiscordCategoryFields) error {
	log.Printf("sync: updating discord category for %q", fields.RoundName)

	// The Discord rate limit on channel renames is fairly restrictive (2 per 10
	// minutes per channel), so finish renaming the category asynchronously if we
	// get rate-limited.
	var name = roundCategoryPrefix + fields.RoundName
	ch := make(chan error)
	go func() {
		ch <- c.discord.SetChannelName(fields.RoundCategory, name)
	}()
	select {
	case err := <-ch:
		return err
	case <-time.After(5 * time.Second):
		rateLimit := c.discord.CheckRateLimit(discordgo.EndpointChannel(fields.RoundCategory))
		if rateLimit == nil {
			// No rate limiting detected; maybe the Discord request is just
			// slow? Wait for it to finish.
			return <-ch
		}
		// Being rate limited; goroutine will finish later.
		msg := fmt.Sprintf(":snail: Hit Discord's rate limit on category renaming. Category will be "+
			"renamed to %q in %s.", name, time.Until(*rateLimit).Round(time.Second))
		_, err := c.discord.ChannelSend(c.discord.QMChannel, msg)
		return err
	}
}

func (c *Client) CheckDiscordRound(ctx context.Context, round state.Round) {
	log.Printf("sync: checking round category for %q", round.Name)
	var original = round.DiscordCategory
	_, err := c.discord.GetChannel(original)
	if discord.ErrCode(err) == discordgo.ErrCodeUnknownChannel {
		created, err := c.CreateDiscordCategory(ctx, round)
		if err != nil {
			return
		}
		go c.state.UpdateRound(ctx, round.ID,
			func(round *state.Round) error {
				if round.DiscordCategory == original {
					log.Printf("sync: replacing deleted discord category for %q", round.Name)
					round.DiscordCategory = created
				} else {
					log.Printf("sync: created duplicate discord category for %q", round.Name)
				}
				return nil
			},
		)
	}
}
