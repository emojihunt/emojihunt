package discord

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

type Config struct {
	QMChannelName, GeneralChannelName, TechChannelName string
	PuzzleCategoryName, SolvedCategoryName             string
}

type Client struct {
	s       *discordgo.Session
	guildID string
	// The QM channel contains a central log of interesting bot actions, as well as the only place for
	// advanced bot usage, such as puzzle or round creation.
	qmChannelID string
	// The general channel has all users, and has announcements from the bot.
	generalChannelID string
	// The tech channel has error messages.
	techChannelID string
	// The puzzle channel category.
	puzzleCategoryID string
	// The category for solved puzzles.
	solvedCategoryID string

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
	ar, ok := chIDs[c.SolvedCategoryName]
	if !ok {
		return nil, fmt.Errorf("archive %q not found", c.SolvedCategoryName)
	}
	gen, ok := chIDs[c.GeneralChannelName]
	if !ok {
		gen = qm
	}
	tech, ok := chIDs[c.TechChannelName]
	if !ok {
		tech = qm
	}

	return &Client{
		s:                s,
		guildID:          guildID,
		qmChannelID:      qm,
		generalChannelID: gen,
		techChannelID:    tech,
		channelNameToID:  chIDs,
		solvedCategoryID: ar,
	}, nil
}

// TODO: Make this a struct with a name.
type NewMessageHandler func(*discordgo.Session, *discordgo.MessageCreate) error

func (c *Client) RegisterNewMessageHandler(name string, h NewMessageHandler) {
	// Only handle new guild messages.
	// TODO: bitOr with the current value.
	c.s.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)
	c.s.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if err := h(s, m); err != nil {
			log.Printf("%s: %v", name, err)
		}
	})
}

// TODO: id or name?
func (c *Client) ChannelURL(id string) string {
	return fmt.Sprintf("https://discord.com/channels/%s/%s", c.guildID, id)
}

func (c *Client) ChannelID(url string) (string, error) {
	if !strings.HasPrefix(url, "https://discord.com/channels/"+c.guildID+"/") {
		return "", fmt.Errorf("invalid channel URL: %q", url)
	}
	parts := strings.Split(url, "/")
	return parts[len(parts)-1], nil
}

func (c *Client) ChannelSend(chanID, msg string) error {
	_, err := c.s.ChannelMessageSend(chanID, msg)
	return err
}

func (c *Client) ChannelSendAndPin(chanID, msg string) error {
	m, err := c.s.ChannelMessageSend(chanID, msg)
	if err != nil {
		return err
	}
	err = c.s.ChannelMessagePin(chanID, m.ID)
	return err
}

func (c *Client) QMChannelSend(msg string) error {
	_, err := c.s.ChannelMessageSend(c.qmChannelID, msg)
	return err
}

func (c *Client) GeneralChannelSend(msg string) error {
	_, err := c.s.ChannelMessageSend(c.generalChannelID, msg)
	return err
}

func (c *Client) TechChannelSend(msg string) error {
	_, err := c.s.ChannelMessageSend(c.techChannelID, msg)
	return err
}

func (c *Client) SolvePuzzle(puzzleName string) error {
	return c.QMChannelSend("hello")
}

var ChannelNotFound = fmt.Errorf("channel not found")

func (c *Client) ArchiveChannel(name string) error {
	c.mu.Lock()
	chID, ok := c.channelNameToID[name]
	c.mu.Unlock()

	if !ok {
		return fmt.Errorf("channel %q not found: %w", name, ChannelNotFound)
	}
	arCh, err := c.s.Channel(c.solvedCategoryID)
	if err != nil {
		return fmt.Errorf("error looking up archive: %v", err)
	}

	_, err = c.s.ChannelEditComplex(chID, &discordgo.ChannelEdit{ParentID: c.solvedCategoryID, PermissionOverwrites: arCh.PermissionOverwrites})
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
	ch, err := c.s.GuildChannelCreateComplex(c.guildID, discordgo.GuildChannelCreateData{
		Name:     name,
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: c.puzzleCategoryID,
	})
	if err != nil {
		return "", fmt.Errorf("error creating channel %q: %v", name, err)
	}
	c.channelNameToID[name] = ch.ID
	return ch.ID, nil
}
