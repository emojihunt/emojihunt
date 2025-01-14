package discord

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"
	"time"

	"github.com/ably/ably-go/ably"
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
	TeamCategoryID      string
}

var DevConfig = Config{
	GuildID:             "1058090773582721214",
	QMRoleID:            "1058092621475614751",
	QMChannelID:         "1058092560926646282",
	HangingOutChannelID: "1058090774488678532",
	MoreEyesChannelID:   "1058092531688157266",
	TeamCategoryID:      "1058090774488678530",
}

var ProdConfig = Config{
	GuildID:             "793599987694436374",
	QMRoleID:            "793618399322046515",
	QMChannelID:         "795780814846689321",
	HangingOutChannelID: "793599987694436377",
	MoreEyesChannelID:   "793607709022748683",
	TeamCategoryID:      "925929203537416293",
}

const ablyChannelName = "discord"

type Client struct {
	main  context.Context
	s     *discordgo.Session
	state *state.Client

	Guild             *discordgo.Guild
	Application       *discordgo.Application
	QMChannel         *discordgo.Channel // for puzzle maintenance
	HangingOutChannel *discordgo.Channel // for solves, to celebrate
	MoreEyesChannel   *discordgo.Channel // for verbose puzzle updates
	TeamCategoryID    string             // for safety
	QMRole            *discordgo.Role    // so QMs show up in the sidebar

	botsByCommand map[string]*botRegistration

	mutex                     sync.Mutex // hold while accessing everything below
	ably                      *ably.RealtimeChannel
	commandsRegistered        bool
	channelCache              map[string]*discordgo.Channel
	memberCache               map[string]*discordgo.Member
	webhookCache              map[string]*discordgo.Webhook
	scheduledEventsCache      map[string]*discordgo.GuildScheduledEvent
	scheduledEventsLastUpdate time.Time
	rateLimits                map[string]*time.Time // url -> retryAfter time
}

func Connect(ctx context.Context, prod bool, state *state.Client, ably *ably.Realtime) *Client {
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
		discordgo.IntentsGuildScheduledEvents |
		discordgo.IntentsGuildMembers |
		discordgo.IntentsGuildMessages |
		discordgo.IntentMessageContent
	s.Identify.Presence.Status = "invisible"
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
		TeamCategoryID:            config.TeamCategoryID,
		QMRole:                    qmRole,
		ably:                      ably.Channels.Get(ablyChannelName),
		botsByCommand:             make(map[string]*botRegistration),
		channelCache:              make(map[string]*discordgo.Channel),
		scheduledEventsLastUpdate: time.Now().Add(-24 * time.Hour),
		rateLimits:                make(map[string]*time.Time),
	}

	// Register handlers. Remember to register the necessary intents above!
	s.AddHandler(WrapHandler(ctx, "bot.unknown", discord.handleCommand))
	s.AddHandler(WrapHandler(ctx, "bot.unknown", discord.handleScheduledEvent))
	s.AddHandler(WrapHandler(ctx, "rate_limit", discord.handleRateLimit))

	s.AddHandler(WrapHandler(ctx, "message", discord.handleMessageCreate))
	s.AddHandler(WrapHandler(ctx, "message", discord.handleMessageUpdate))
	s.AddHandler(WrapHandler(ctx, "message", discord.handleMessageDelete))

	s.AddHandler(WrapHandler(ctx, "member", discord.handleGuildMemberAdd))
	s.AddHandler(WrapHandler(ctx, "member", discord.handleGuildMemberUpdate))
	s.AddHandler(WrapHandler(ctx, "member", discord.handleGuildMemberRemove))

	s.AddHandler(WrapHandler(ctx, "channel", discord.handleChannelCreate))
	s.AddHandler(WrapHandler(ctx, "channel", discord.handleChannelUpdate))
	s.AddHandler(WrapHandler(ctx, "channel", discord.handleChannelDelete))

	if err := discord.refreshChannelCache(); err != nil {
		log.Panicf("refreshChannelCache: %v", err)
	}
	if err := discord.refreshMemberCache(); err != nil {
		log.Panicf("refreshMemberCache: %v", err)
	}
	if err := discord.refreshWebhookCache(); err != nil {
		log.Panicf("refreshMemberCache: %v", err)
	}
	return discord
}

func ErrCode(err error) int {
	var cast *discordgo.RESTError
	if errors.As(err, &cast) && cast.Message != nil {
		return cast.Message.Code
	}
	return 0
}

func (c *Client) Close() error {
	return c.s.Close()
}

func (c *Client) UpdateStatus(data discordgo.UpdateStatusData) error {
	return c.s.UpdateStatusComplex(data)
}

func (c *Client) ChannelSend(ch *discordgo.Channel, msg string) (string, error) {
	if len(msg) > 2000 {
		msg = msg[:1994] + "\n[...]"
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

// Set the pinned status message, by posting one or editing the existing one.
// No-op if the status was already set.
func (c *Client) CreateUpdatePin(chanID string, embed *discordgo.MessageEmbed) error {
	existing, err := c.s.ChannelMessagesPinned(chanID)
	if err != nil {
		return xerrors.Errorf("ChannelMessagesPinned: %w", err)
	}
	var statusMessage *discordgo.Message
	for _, msg := range existing {
		if len(msg.Embeds) > 0 {
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

func (c *Client) GetTopReaction(channel *discordgo.Channel, messageID string) (string, error) {
	msg, err := c.GetMessage(channel, messageID)
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
