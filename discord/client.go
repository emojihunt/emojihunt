package discord

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/state"
	"golang.org/x/xerrors"
)

type Config struct {
	GuildID             string
	QMRoleID            string
	QMChannelID         string
	HangingOutChannelID string
	MoreEyesChannelID   string
}

var DevConfig = Config{
	GuildID:             "1058090773582721214",
	QMRoleID:            "1058092621475614751",
	QMChannelID:         "1058092560926646282",
	HangingOutChannelID: "1058090774488678532",
	MoreEyesChannelID:   "1058092531688157266",
}

var ProdConfig = Config{
	GuildID:             "793599987694436374",
	QMRoleID:            "793618399322046515",
	QMChannelID:         "795780814846689321",
	HangingOutChannelID: "793599987694436377",
	MoreEyesChannelID:   "793607709022748683",
}

type Client struct {
	main  context.Context
	s     *discordgo.Session
	state *state.Client

	Guild               *discordgo.Guild
	Application         *discordgo.Application
	QMChannel           *discordgo.Channel // for puzzle maintenance
	HangingOutChannel   *discordgo.Channel // for solves, to celebrate
	MoreEyesChannel     *discordgo.Channel // for verbose puzzle updates
	DefaultVoiceChannel *discordgo.Channel // for placeholder events
	QMRole              *discordgo.Role    // so QMs show up in the sidebar

	botsByCommand map[string]*botRegistration

	mu                        sync.Mutex // hold while accessing everything below
	commandsRegistered        bool
	scheduledEventsCache      map[string]*discordgo.GuildScheduledEvent
	scheduledEventsLastUpdate time.Time
	rateLimits                map[string]*time.Time // url -> retryAfter time
	voiceRooms                map[string]*discordgo.Channel
}

func Connect(ctx context.Context, prod bool, state *state.Client) *Client {
	// Initialize discordgo client
	token, ok := os.LookupEnv("DISCORD_TOKEN")
	if !ok {
		panic("DISCORD_TOKEN is required")
	}
	s, err := discordgo.New(token)
	if err != nil {
		panic(err)
	}
	s.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildScheduledEvents
	s.Identify.Presence.Status = computeBotStatus(ctx, state)
	if err := s.Open(); err != nil {
		log.Panicf("discordgo.Open: %s", err)
	}

	// Validate config
	var config = DevConfig
	if prod {
		config = ProdConfig
	}
	guild, err := s.Guild(config.GuildID)
	if err != nil {
		log.Panicf("failed to load guild %s: %s", config.GuildID, err)
	}

	app, err := s.Application("@me")
	if err != nil {
		log.Panicf("failed to load application @me: %s", err)
	}

	hangingOutChannel, err := s.Channel(config.HangingOutChannelID)
	if err != nil {
		log.Panicf("failed to load hanging-out channel %q: %s",
			config.HangingOutChannelID, err)
	}
	moreEyesChannel, err := s.Channel(config.MoreEyesChannelID)
	if err != nil {
		log.Panicf("failed to load more-eyes channel %q: %s",
			config.MoreEyesChannelID, err)
	}
	qmChannel, err := s.Channel(config.QMChannelID)
	if err != nil {
		log.Panicf("failed to load qm channel %q: %s", config.QMChannelID, err)
	}

	var defaultVoiceChannel *discordgo.Channel
	channels, err := s.GuildChannels(config.GuildID)
	if err != nil {
		log.Panicf("failed to load voice channels: %s", err)
	}
	for _, channel := range channels {
		if channel.Type == discordgo.ChannelTypeGuildVoice {
			defaultVoiceChannel = channel
			break
		}
	}
	if defaultVoiceChannel == nil {
		log.Panicf("no voice channels found")
	}

	allRoles, err := s.GuildRoles(guild.ID)
	if err != nil {
		log.Panicf("failed to load guild roles: %s", err)
	}
	var qmRole *discordgo.Role
	for _, role := range allRoles {
		if role.ID == config.QMRoleID {
			qmRole = role
		}
	}
	if qmRole == nil {
		log.Panicf("role %q not found in guild %q", config.QMRoleID, guild.ID)
	}

	// Set up slash commands; return
	discord := &Client{
		main:                      ctx,
		s:                         s,
		state:                     state,
		Guild:                     guild,
		Application:               app,
		HangingOutChannel:         hangingOutChannel,
		MoreEyesChannel:           moreEyesChannel,
		QMChannel:                 qmChannel,
		DefaultVoiceChannel:       defaultVoiceChannel,
		QMRole:                    qmRole,
		botsByCommand:             make(map[string]*botRegistration),
		scheduledEventsLastUpdate: time.Now().Add(-24 * time.Hour),
		rateLimits:                make(map[string]*time.Time),
		voiceRooms:                make(map[string]*discordgo.Channel),
	}

	// Register handlers. Remember to register the necessary intents above!
	s.AddHandler(WrapHandler(ctx, "bot.unknown", discord.handleCommand))
	s.AddHandler(WrapHandler(ctx, "bot.unknown", discord.handleScheduledEvent))
	s.AddHandler(WrapHandler(ctx, "rate_limit", discord.handleRateLimit))

	s.AddHandler(WrapHandler(ctx, "channel", discord.handleChannelCreate))
	s.AddHandler(WrapHandler(ctx, "channel", discord.handleChannelUpdate))
	s.AddHandler(WrapHandler(ctx, "channel", discord.handleChannelDelete))
	if err := discord.refreshVoiceChannels(); err != nil {
		log.Panicf("failed to refresh voice channels: %s", err)
	}

	return discord
}

func (c *Client) Close() error {
	return c.s.Close()
}

// Update the bot's status (idle/active).
func (c *Client) UpdateStatus(ctx context.Context) error {
	return c.s.UpdateStatusComplex(discordgo.UpdateStatusData{
		Status: computeBotStatus(ctx, c.state),
	})
}

func computeBotStatus(ctx context.Context, state *state.Client) string {
	if state.IsEnabled(ctx) {
		return "online"
	} else {
		return "dnd"
	}
}

func (c *Client) ChannelSend(ch *discordgo.Channel, msg string) (string, error) {
	if len(msg) > 1950 {
		msg = msg[:1950] + "... [truncated]"
	}
	sent, err := c.s.ChannelMessageSend(ch.ID, msg)
	if err != nil {
		return "", xerrors.Errorf("ChannelMessageSend: %w", err)
	}
	return sent.ID, nil
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
		return "", xerrors.Errorf("ChannelMessageSendComplex: %w", err)
	}
	return sent.ID, nil
}

func (c *Client) ChannelSendRawID(chID, msg string) error {
	if _, err := c.s.ChannelMessageSend(chID, msg); err != nil {
		return xerrors.Errorf("ChannelMessageSend: %w", err)
	}
	return nil
}

func (c *Client) GetMessage(ch *discordgo.Channel, messageID string) (*discordgo.Message, error) {
	msg, err := c.s.ChannelMessage(ch.ID, messageID)
	if err != nil {
		return nil, xerrors.Errorf("ChannelMessage: %w", err)
	}
	return msg, nil
}

func (c *Client) CreateChannel(name string, category *discordgo.Channel) (*discordgo.Channel, error) {
	ch, err := c.s.GuildChannelCreateComplex(c.Guild.ID, discordgo.GuildChannelCreateData{
		Name:     name,
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: category.ID,
	})
	if err != nil {
		return nil, xerrors.Errorf("GuildChannelCreateComplex: %w", err)
	}
	return ch, nil
}

func (c *Client) SetChannelName(chID, name string) error {
	_, err := c.s.ChannelEdit(chID, &discordgo.ChannelEdit{
		Name: name,
	})
	if err != nil {
		return xerrors.Errorf("ChannelEdit: %w", err)
	}
	return nil
}

func (c *Client) ChannelValue(chOpt *discordgo.ApplicationCommandInteractionDataOption) *discordgo.Channel {
	return chOpt.ChannelValue(c.s)
}

func (c *Client) GetChannelCategories() (map[string]*discordgo.Channel, error) {
	channels, err := c.s.GuildChannels(c.Guild.ID)
	if err != nil {
		return nil, xerrors.Errorf("GuildChannels: %w", err)
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
		return xerrors.Errorf("channel id %s not found", chID)
	}

	if ch.ParentID == category.ID {
		return nil // no-op
	}

	_, err = c.s.ChannelEditComplex(chID, &discordgo.ChannelEdit{
		ParentID:             category.ID,
		PermissionOverwrites: category.PermissionOverwrites,
	})
	if err != nil {
		return xerrors.Errorf("error moving channel to category %q: %w", category.Name, err)
	}
	return nil
}

// Set the pinned status message, by posting one or editing the existing one.
// No-op if the status was already set.
func (c *Client) CreateUpdatePin(chanID, header string, embed *discordgo.MessageEmbed) error {
	existing, err := c.s.ChannelMessagesPinned(chanID)
	if err != nil {
		return xerrors.Errorf("ChannelMessagesPinned: %w", err)
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
			return xerrors.Errorf("ChannelMessageSendEmbed: %w", err)
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

func (c *Client) GetGuildMember(user *discordgo.User) (*discordgo.Member, error) {
	member, err := c.s.GuildMember(c.Guild.ID, user.ID)
	if err != nil {
		return nil, xerrors.Errorf("GuildMember: %w", err)
	}
	return member, nil
}

func (c *Client) MakeQM(user *discordgo.User) error {
	err := c.s.GuildMemberRoleAdd(c.Guild.ID, user.ID, c.QMRole.ID)
	if err != nil {
		return xerrors.Errorf("GuildMemberRoleAdd: %w", err)
	}
	return nil
}

func (c *Client) UnMakeQM(user *discordgo.User) error {
	err := c.s.GuildMemberRoleRemove(c.Guild.ID, user.ID, c.QMRole.ID)
	if err != nil {
		return xerrors.Errorf("GuildMemberRoleRemove: %w", err)
	}
	return nil
}
