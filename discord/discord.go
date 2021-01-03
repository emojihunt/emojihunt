package discord

import (
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
)

type Config struct {
	QMChannelName, GeneralChannelName, ArchiveChannelName string
}

type Client struct {
	s       *discordgo.Session
	guildID string
	// The QM channel contains a central log of interesting bot actions, as well as the only place for
	// advanced bot usage, such as puzzle or round creation.
	qmChannelID string
	// The general channel has all users, and has announcements from the bot.
	generalChannelID string
	// archive is a category, ie. a discord channel that can have children.
	archiveID string

	// This might be a case where sync.Map makes sense.
	mu              sync.Mutex
	channelNameToID map[string]string
}

func getGuildID(s *discordgo.Session) (string, error) {
	gs := s.State.Guilds
	if len(gs) != 1 {
		return "", fmt.Errorf("expected exactly 1 guild, found %d", len(gs))
	}
	return gs[0].ID, nil
}

func New(s *discordgo.Session, c Config) (*Client, error) {
	guildID, err := getGuildID(s)
	if err != nil {
		return nil, err
	}
	chs, err := s.GuildChannels(guildID)
	if err != nil {
		return nil, fmt.Errorf("error creating channel ID cache: %v", err)
	}
	chIDs := make(map[string]string)
	for _, ch := range chs {
		chIDs[ch.Name] = ch.ID
	}
	qm, ok := chIDs[c.QMChannelName]
	if !ok {
		return nil, fmt.Errorf("QM Channel %q not found", c.QMChannelName)
	}
	ar, ok := chIDs[c.ArchiveChannelName]
	if !ok {
		return nil, fmt.Errorf("archive %q not found", c.ArchiveChannelName)
	}
	gen, ok := chIDs[c.GeneralChannelName]
	if !ok {
		gen = qm
	}

	return &Client{
		s:                s,
		guildID:          guildID,
		qmChannelID:      qm,
		generalChannelID: gen,
		channelNameToID:  chIDs,
		archiveID:        ar,
	}, nil
}

func (c *Client) RegisterNewMessageHandler(h func(*discordgo.Session, *discordgo.MessageCreate)) {
	// Only handle new guild messages.
	// TODO: bitOr with the current value.
	c.s.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)
	c.s.AddHandler(h)
}

func (c *Client) QMChannelSend(msg string) error {
	_, err := c.s.ChannelMessageSend(c.qmChannelID, msg)
	return err
}

func (c *Client) GeneralChannelSend(msg string) error {
	_, err := c.s.ChannelMessageSend(c.generalChannelID, msg)
	return err
}

func (c *Client) SolvePuzzle(puzzleName string) error {
	return c.QMChannelSend("hello")
}

type ChannelNotFoundError string

func (e ChannelNotFoundError) Error() string {
	return fmt.Sprintf("channel %q not found", string(e))
}

func (c *Client) ArchiveChannel(name string) error {
	c.mu.Lock()
	chID, ok := c.channelNameToID[name]
	c.mu.Unlock()

	if !ok {
		return ChannelNotFoundError(name)
	}
	arCh, err := c.s.Channel(c.archiveID)
	if err != nil {
		return fmt.Errorf("error looking up archive: %v", err)
	}

	_, err = c.s.ChannelEditComplex(chID, &discordgo.ChannelEdit{ParentID: c.archiveID, PermissionOverwrites: arCh.PermissionOverwrites})
	if err != nil {
		return fmt.Errorf("error moving channel: %v", err)
	}
	return err
}

// CreateChannel ensures that a channel exists with the given name, and returns the channel ID.
func (c *Client) CreateChannel(name string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if id, ok := c.channelNameToID[name]; ok {
		return id, nil
	}
	ch, err := c.s.GuildChannelCreate(c.guildID, name, discordgo.ChannelTypeGuildText)
	if err != nil {
		return "", fmt.Errorf("error creating channel %q: %v", name, err)
	}
	c.channelNameToID[name] = ch.ID
	return ch.ID, nil
}
