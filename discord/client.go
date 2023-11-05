package discord

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/state"
)

type Config struct {
	AuthToken           string `json:"auth_token"`
	IssueURL            string `json:"issue_url"`
	GuildID             string `json:"guild_id"`
	QMChannelID         string `json:"qm_channel_id"`
	HangingOutChannelID string `json:"hanging_out_channel_id"`
	MoreEyesChannelID   string `json:"more_eyes_channel_id"`
	TechChannelID       string `json:"tech_channel_id"`
	QMRoleID            string `json:"qm_role_id"`
}

type Client struct {
	main     context.Context
	issueURL string

	s     *discordgo.Session
	Guild *discordgo.Guild

	HangingOutChannel *discordgo.Channel // for solves, to celebrate
	MoreEyesChannel   *discordgo.Channel // for verbose puzzle updates
	QMChannel         *discordgo.Channel // for puzzle maintenance
	TechChannel       *discordgo.Channel // for error messages

	DefaultVoiceChannel *discordgo.Channel // for placeholder events

	QMRole *discordgo.Role // so QMs show up in the sidebar

	commandHandlers  map[string]*botRegistration
	eventHandlers    []*func(context.Context, *discordgo.GuildScheduledEventUpdate) error
	reactionHandlers []*func(context.Context, *discordgo.MessageReaction) error

	mu                        sync.Mutex // hold while accessing everything below
	commandsRegistered        bool
	scheduledEventsCache      map[string]*discordgo.GuildScheduledEvent
	scheduledEventsLastUpdate time.Time
	rateLimits                map[string]*time.Time // url -> retryAfter time
}

func Connect(ctx context.Context, config *Config, state *state.State) (*Client, error) {
	// Initialize discordgo client
	s, err := discordgo.New(config.AuthToken)
	if err != nil {
		return nil, err
	}
	s.Identify.Intents = discordgo.IntentsGuildMessageReactions |
		discordgo.IntentsGuildScheduledEvents

	state.Lock()
	s.Identify.Presence.Status = computeBotStatus(state)
	state.Unlock()
	if err := s.Open(); err != nil {
		return nil, err
	}

	// Validate config
	guild, err := s.Guild(config.GuildID)
	if err != nil {
		return nil, fmt.Errorf("failed to load guild %s: %w", config.GuildID, err)
	}

	hangingOutChannel, err := s.Channel(config.HangingOutChannelID)
	if err != nil {
		return nil, fmt.Errorf("failed to load hanging-out channel %q: %w",
			config.HangingOutChannelID, err)
	}
	moreEyesChannel, err := s.Channel(config.MoreEyesChannelID)
	if err != nil {
		return nil, fmt.Errorf("failed to load more-eyes channel %q: %w",
			config.MoreEyesChannelID, err)
	}
	qmChannel, err := s.Channel(config.QMChannelID)
	if err != nil {
		return nil, fmt.Errorf("failed to load qm channel %q: %w",
			config.QMChannelID, err)
	}
	techChannel, err := s.Channel(config.TechChannelID)
	if err != nil {
		return nil, fmt.Errorf("failed to load tech channel %q: %w",
			config.TechChannelID, err)
	}

	var defaultVoiceChannel *discordgo.Channel
	channels, err := s.GuildChannels(config.GuildID)
	if err != nil {
		return nil, fmt.Errorf("failed to load voice channels: %w", err)
	}
	for _, channel := range channels {
		if channel.Type == discordgo.ChannelTypeGuildVoice {
			defaultVoiceChannel = channel
			break
		}
	}
	if defaultVoiceChannel == nil {
		return nil, fmt.Errorf("no voice channels found")
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
	discord := &Client{
		main:                      ctx,
		issueURL:                  config.IssueURL,
		s:                         s,
		Guild:                     guild,
		HangingOutChannel:         hangingOutChannel,
		MoreEyesChannel:           moreEyesChannel,
		QMChannel:                 qmChannel,
		TechChannel:               techChannel,
		DefaultVoiceChannel:       defaultVoiceChannel,
		QMRole:                    qmRole,
		commandHandlers:           make(map[string]*botRegistration),
		scheduledEventsLastUpdate: time.Now().Add(-24 * time.Hour),
		rateLimits:                make(map[string]*time.Time),
	}

	// Register handlers. Remember to register the necessary intents above!
	s.AddHandler(WrapHandler(ctx, "bot.unknown", discord.handleCommand))
	s.AddHandler(WrapHandler(ctx, "event", discord.handleScheduledEvent))
	s.AddHandler(WrapHandler(ctx, "reaction",
		func(ctx context.Context, r *discordgo.MessageReactionAdd) error {
			return discord.handleReaction(ctx, r.MessageReaction)
		}),
	)
	s.AddHandler(WrapHandler(ctx, "reaction",
		func(ctx context.Context, r *discordgo.MessageReactionRemove) error {
			return discord.handleReaction(ctx, r.MessageReaction)
		}),
	)
	s.AddHandler(WrapHandler(ctx, "reaction",
		func(ctx context.Context, r *discordgo.MessageReactionRemoveAll) error {
			return discord.handleReaction(ctx, r.MessageReaction)
		}),
	)
	s.AddHandler(WrapHandler(ctx, "rate_limit", discord.handleRateLimit))

	return discord, nil
}

func (c *Client) Close() error {
	return c.s.Close()
}

// Update the bot's status (idle/active). The caller must hold the state lock.
func (c *Client) UpdateStatus(state *state.State) error {
	return c.s.UpdateStatusComplex(discordgo.UpdateStatusData{
		Status: computeBotStatus(state),
	})
}

// Caller must hold the state lock
func computeBotStatus(state *state.State) string {
	if state.HuntbotDisabled {
		return "dnd"
	} else {
		return "online"
	}
}

func (c *Client) ChannelSend(ch *discordgo.Channel, msg string) (string, error) {
	if len(msg) > 1950 {
		msg = msg[:1950] + "... [truncated]"
	}
	sent, err := c.s.ChannelMessageSend(ch.ID, msg)
	if err != nil {
		return "", err
	}
	return sent.ID, err
}

func (c *Client) ChannelSendComponents(ch *discordgo.Channel, msg string,
	components []discordgo.MessageComponent) (string, error) {

	var actionsRow []discordgo.MessageComponent
	if components != nil {
		actionsRow = []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: components,
			},
		}
	}
	sent, err := c.s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
		Content:    msg,
		Components: actionsRow,
	})
	if err != nil {
		return "", err
	}
	return sent.ID, nil
}

func (c *Client) ChannelSendRawID(chID, msg string) error {
	_, err := c.s.ChannelMessageSend(chID, msg)
	return err
}

func (c *Client) GetMessage(ch *discordgo.Channel, messageID string) (*discordgo.Message, error) {
	return c.s.ChannelMessage(ch.ID, messageID)
}

func (c *Client) CreateChannel(name string, category *discordgo.Channel) (*discordgo.Channel, error) {
	return c.s.GuildChannelCreateComplex(c.Guild.ID, discordgo.GuildChannelCreateData{
		Name:     name,
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: category.ID,
	})
}

func (c *Client) SetChannelName(chID, name string) error {
	_, err := c.s.ChannelEdit(chID, &discordgo.ChannelEdit{
		Name: name,
	})
	return err
}

func (c *Client) ChannelValue(chOpt *discordgo.ApplicationCommandInteractionDataOption) *discordgo.Channel {
	return chOpt.ChannelValue(c.s)
}

func (c *Client) GetChannelCategories() (map[string]*discordgo.Channel, error) {
	channels, err := c.s.GuildChannels(c.Guild.ID)
	if err != nil {
		return nil, err
	}

	var categories = make(map[string]*discordgo.Channel)
	for _, channel := range channels {
		if channel.Type == discordgo.ChannelTypeGuildCategory {
			categories[channel.Name] = channel
		}
	}
	return categories, nil
}

func (c *Client) CreateCategory(name string) (*discordgo.Channel, error) {
	return c.s.GuildChannelCreate(c.Guild.ID, name, discordgo.ChannelTypeGuildCategory)
}

func (c *Client) SetChannelCategory(chID string, category *discordgo.Channel) error {
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
		return fmt.Errorf("error moving channel to category %q: %w", category.Name, err)
	}
	return nil
}

// Set the pinned status message, by posting one or editing the existing one.
// No-op if the status was already set.
func (c *Client) CreateUpdatePin(chanID, header string, embed *discordgo.MessageEmbed) error {
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

func (c *Client) MakeQM(user *discordgo.User) error {
	return c.s.GuildMemberRoleAdd(c.Guild.ID, user.ID, c.QMRole.ID)
}

func (c *Client) UnMakeQM(user *discordgo.User) error {
	return c.s.GuildMemberRoleRemove(c.Guild.ID, user.ID, c.QMRole.ID)
}
