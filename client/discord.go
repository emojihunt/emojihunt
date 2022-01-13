package client

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

type DiscordConfig struct {
	AuthToken         string `json:"auth_token"`
	GuildID           string `json:"guild_id"`
	QMChannelID       string `json:"qm_channel_id"`
	KitchenChannelID  string `json:"kitchen_channel_id"`
	MoreEyesChannelID string `json:"more_eyes_channel_id"`
	TechChannelID     string `json:"tech_channel_id"`
	PuzzleCategoryID  string `json:"puzzle_category_id"`
	SolvedCategoryID  string `json:"solved_category_id"`
	QMRoleID          string `json:"qm_role_id"`
}

type Discord struct {
	s     *discordgo.Session
	Guild *discordgo.Guild

	KitchenChannel  *discordgo.Channel // for solves, to celebrate
	MoreEyesChannel *discordgo.Channel // for verbose puzzle updates
	QMChannel       *discordgo.Channel // for puzzle maintenance
	TechChannel     *discordgo.Channel // for error messages

	PuzzleCategory *discordgo.Channel // for unsolved puzzles
	SolvedCategory *discordgo.Channel // for solved puzzles

	QMRole *discordgo.Role // so QMs show up in the sidebar

	appCommandHandlers map[string]*DiscordCommand
	componentHandlers  map[string]*DiscordCommand

	mu                        sync.Mutex // hold while accessing everything below
	commandsRegistered        bool
	scheduledEventsCache      map[string]*discordgo.GuildScheduledEvent
	scheduledEventsLastUpdate time.Time
	rateLimits                map[string]*time.Time // url -> retryAfter time
}

func NewDiscord(config *DiscordConfig) (*Discord, error) {
	// Initialize discordgo client
	s, err := discordgo.New(config.AuthToken)
	if err != nil {
		return nil, err
	}
	// s.Debug = true // warning: it's *very* verbose
	s.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildScheduledEvents
	if err := s.Open(); err != nil {
		return nil, err
	}

	// Validate config
	guild, err := s.Guild(config.GuildID)
	if err != nil {
		return nil, fmt.Errorf("failed to load guild %s: %v", config.GuildID, err)
	}
	kitchenChannel, err := s.Channel(config.KitchenChannelID)
	if err != nil {
		return nil, fmt.Errorf("failed to load kitchen channel %q: %v",
			config.KitchenChannelID, err)
	}
	moreEyesChannel, err := s.Channel(config.MoreEyesChannelID)
	if err != nil {
		return nil, fmt.Errorf("failed to load more eyes channel %q: %v",
			config.MoreEyesChannelID, err)
	}
	qmChannel, err := s.Channel(config.QMChannelID)
	if err != nil {
		return nil, fmt.Errorf("failed to load qm channel %q: %v",
			config.QMChannelID, err)
	}
	techChannel, err := s.Channel(config.TechChannelID)
	if err != nil {
		return nil, fmt.Errorf("failed to load tech channel %q: %v",
			config.TechChannelID, err)
	}
	puzzleCategory, err := s.Channel(config.PuzzleCategoryID)
	if err != nil {
		return nil, err
	} else if puzzleCategory.Type != discordgo.ChannelTypeGuildCategory {
		return nil, fmt.Errorf("puzzle category is wrong type: %v", puzzleCategory.Type)
	}
	solvedCategory, err := s.Channel(config.SolvedCategoryID)
	if err != nil {
		return nil, err
	} else if puzzleCategory.Type != discordgo.ChannelTypeGuildCategory {
		return nil, fmt.Errorf("solved category is wrong type: %v", puzzleCategory.Type)
	}
	allRoles, err := s.GuildRoles(guild.ID)
	if err != nil {
		return nil, err
	}
	var qmRole *discordgo.Role
	for _, role := range allRoles {
		if role.ID == config.QMRoleID {
			qmRole = role
		}
	}
	if qmRole == nil {
		return nil, fmt.Errorf("role %q not found in guild %q", config.QMRoleID, guild.ID)
	}

	// Set up slash commands; return
	discord := &Discord{
		s:                         s,
		Guild:                     guild,
		KitchenChannel:            kitchenChannel,
		MoreEyesChannel:           moreEyesChannel,
		QMChannel:                 qmChannel,
		TechChannel:               techChannel,
		PuzzleCategory:            puzzleCategory,
		SolvedCategory:            solvedCategory,
		QMRole:                    qmRole,
		appCommandHandlers:        make(map[string]*DiscordCommand),
		componentHandlers:         make(map[string]*DiscordCommand),
		scheduledEventsLastUpdate: time.Now().Add(-24 * time.Hour),
		rateLimits:                make(map[string]*time.Time),
	}
	s.AddHandler(discord.commandHandler)
	s.AddHandler(func(s *discordgo.Session, r *discordgo.RateLimit) {
		log.Printf("discord: hit rate limit at %q: %#v", r.URL, r.TooManyRequests)
		expiry := time.Now().Add(r.TooManyRequests.RetryAfter)

		discord.mu.Lock()
		defer discord.mu.Unlock()
		discord.rateLimits[r.URL] = &expiry
	})
	return discord, nil
}

func (c *Discord) CheckRateLimit(url string) *time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	limit := c.rateLimits[url]
	if limit == nil || time.Now().After(*limit) {
		return nil
	}
	return limit
}

func (c *Discord) AddHandler(handler interface{}) {
	// Remember to add your intent type to the Intents assignment above!
	c.s.AddHandler(handler)
}

func (c *Discord) Close() error {
	return c.s.Close()
}

func (c *Discord) ChannelSend(ch *discordgo.Channel, msg string) error {
	_, err := c.s.ChannelMessageSend(ch.ID, msg)
	return err
}

func (c *Discord) ChannelSendEmbed(ch *discordgo.Channel, embed *discordgo.MessageEmbed) error {
	_, err := c.s.ChannelMessageSendEmbed(ch.ID, embed)
	return err
}

func (c *Discord) ChannelSendComponents(ch *discordgo.Channel, msg string, components []discordgo.MessageComponent) error {
	var actionsRow []discordgo.MessageComponent
	if components != nil {
		actionsRow = []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: components,
			},
		}
	}
	_, err := c.s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
		Content:    msg,
		Components: actionsRow,
	})
	return err
}

func (c *Discord) ChannelSendRawID(chID, msg string) error {
	_, err := c.s.ChannelMessageSend(chID, msg)
	return err
}

func (c *Discord) CreateChannel(name string) (*discordgo.Channel, error) {
	return c.s.GuildChannelCreateComplex(c.Guild.ID, discordgo.GuildChannelCreateData{
		Name:     name,
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: c.PuzzleCategory.ID,
	})
}

func (c *Discord) SetChannelName(chID, name string) error {
	_, err := c.s.ChannelEdit(chID, name)
	return err
}

func (c *Discord) SetChannelCategory(chID string, category *discordgo.Channel) error {
	ch, err := c.s.Channel(chID)
	if err != nil {
		return fmt.Errorf("channel id %s not found", chID)
	}

	if ch.ParentID == category.ID {
		return nil // no-op
	}

	_, err = c.s.ChannelEditComplex(chID, &discordgo.ChannelEdit{
		ParentID:             category.ID,
		PermissionOverwrites: category.PermissionOverwrites,
	})
	if err != nil {
		return fmt.Errorf("error moving channel to category %q: %v", category.Name, err)
	}
	return nil
}

// Set the pinned status message, by posting one or editing the existing one.
// No-op if the status was already set.
func (c *Discord) CreateUpdatePin(chanID, header string, embed *discordgo.MessageEmbed) error {
	existing, err := c.s.ChannelMessagesPinned(chanID)
	if err != nil {
		return err
	}
	var statusMessage *discordgo.Message
	for _, msg := range existing {
		if len(msg.Embeds) > 0 && msg.Embeds[0].Author.Name == header {
			if statusMessage != nil {
				log.Printf("discord: multiple status messages in %v, editing last one", chanID)
			}
			statusMessage = msg
		}
	}

	if statusMessage == nil {
		// create a pinned message
		m, err := c.s.ChannelMessageSendEmbed(chanID, embed)
		if err != nil {
			return err
		}
		return c.s.ChannelMessagePin(chanID, m.ID)
	} else {
		// update existing pinned message
		if statusMessage.Embeds[0] == embed {
			return nil // no-op
		}
		_, err = c.s.ChannelMessageEditEmbed(chanID, statusMessage.ID, embed)
		return err
	}
}
