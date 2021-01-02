package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type Config struct {
	QMChannelName, ArchiveChannelName string
}

type Client struct {
	s       *discordgo.Session
	guildID string
	// TODO: not a great idea to keep these channels around, could lead to inconsistency
	qmChannelID     string
	channelNameToID map[string]string
	// archive is a category, ie. a discord channel that can have children.
	archiveID string
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

	return &Client{
		s:               s,
		guildID:         guildID,
		qmChannelID:     qm,
		channelNameToID: chIDs,
		archiveID:       ar,
	}, nil
}

type NewMessageHandler func(*discordgo.Session, *discordgo.MessageCreate)

func (c *Client) RegisterNewMessageHandler(h NewMessageHandler) {
	// Only handle new guild messages.
	// TODO: bitor with the current value.
	c.s.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)
	c.s.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) { h(s, m) })
}

func (c *Client) QMChannelSend(msg string) error {
	_, err := c.s.ChannelMessageSend(c.qmChannelID, msg)
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
	chID, ok := c.channelNameToID[name]
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

func (c *Client) CreatePuzzle() {
}
