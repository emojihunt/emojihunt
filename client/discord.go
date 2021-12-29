package client

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

type DiscordConfig struct {
	GuildID                                                                     string
	QMChannelName, GeneralChannelName, StatusUpdateChannelName, TechChannelName string
	PuzzleCategoryName, SolvedCategoryName                                      string
	QMRoleName                                                                  string
}

type Discord struct {
	s       *discordgo.Session
	GuildID string
	// The QM channel contains a central log of interesting bot actions, as well as the only place for
	// advanced bot usage, such as puzzle or round creation.
	QMChannelID string
	// The general channel has all users, and has announcements from the bot.
	generalChannelID string
	// The channel in which to post status updates.
	statusUpdateChannelID string
	// The tech channel has error messages.
	techChannelID string
	// The puzzle channel category.
	puzzleCategoryID string
	// The category for solved puzzles.
	solvedCategoryID string
	// The Role ID for the QM role.
	QMRoleID string

	// This might be a case where sync.Map makes sense.
	mu              sync.Mutex
	channelNameToID map[string]string
	// A room is the common name of a voice channel, excluding the puzzles present there.
	// For example, a voice channel "Patio: puzzle 1, puzzle 2", is named "Patio", and has 2 puzzles being worked on.
	roomsToID map[string]string
}

func NewDiscord(s *discordgo.Session, c DiscordConfig) (*Discord, error) {
	if c.GuildID == "" {
		var err error
		c.GuildID, err = getGuildID(s)
		if err != nil {
			return nil, err
		}
	}
	chs, err := s.GuildChannels(c.GuildID)
	if err != nil {
		return nil, fmt.Errorf("error creating channel ID cache: %v", err)
	}
	chIDs := make(map[string]string)
	rIDs := make(map[string]string)
	for _, ch := range chs {
		chIDs[ch.Name] = ch.ID
		if ch.Bitrate != 0 {
			r, err := parseRoom(ch.Name)
			if err != nil {
				return nil, err
			}
			rIDs[r.name] = ch.ID
		}
	}
	qm, ok := chIDs[c.QMChannelName]
	if !ok {
		return nil, fmt.Errorf("QM Channel %q not found", c.QMChannelName)
	}
	puz, ok := chIDs[c.PuzzleCategoryName]
	if !ok {
		return nil, fmt.Errorf("puzzle category %q not found in channels: %v", c.PuzzleCategoryName, chIDs)
	}
	ar, ok := chIDs[c.SolvedCategoryName]
	if !ok {
		return nil, fmt.Errorf("archive %q not found", c.SolvedCategoryName)
	}
	gen, ok := chIDs[c.GeneralChannelName]
	if !ok {
		gen = qm
	}
	st, ok := chIDs[c.StatusUpdateChannelName]
	if !ok {
		st = gen
	}
	tech, ok := chIDs[c.TechChannelName]
	if !ok {
		tech = qm
	}
	roles, err := s.GuildRoles(c.GuildID)
	if err != nil {
		return nil, fmt.Errorf("error fetching roles: %v", err)
	}
	var qmRoleID string
	for _, r := range roles {
		if r.Name == c.QMRoleName {
			qmRoleID = r.ID
			break
		}
	}
	if qmRoleID == "" {
		return nil, fmt.Errorf("QM role %q not found in roles: %v", c.QMRoleName, roles)
	}

	return &Discord{
		s:                     s,
		GuildID:               c.GuildID,
		QMChannelID:           qm,
		generalChannelID:      gen,
		statusUpdateChannelID: st,
		techChannelID:         tech,
		channelNameToID:       chIDs,
		roomsToID:             rIDs,
		puzzleCategoryID:      puz,
		solvedCategoryID:      ar,
		QMRoleID:              qmRoleID,
	}, nil
}

func getGuildID(s *discordgo.Session) (string, error) {
	log.Print("fetching GuildID from session")
	gs := s.State.Guilds
	if len(gs) != 1 {
		return "", fmt.Errorf("expected exactly 1 guild, found %d", len(gs))
	}
	return gs[0].ID, nil
}

type DiscordMessageHandler func(*discordgo.Session, *discordgo.MessageCreate) error

func (c *Discord) RegisterNewMessageHandler(name string, h DiscordMessageHandler) {
	// Only handle new guild messages.
	// TODO: bitOr with the current value.
	c.s.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)
	c.s.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if err := h(s, m); err != nil {
			log.Printf("%s: %v", name, err)
		}
	})
}

func (c *Discord) ChannelSend(chanID, msg string) error {
	_, err := c.s.ChannelMessageSend(chanID, msg)
	return err
}

// Returns last pinned status message, or nil if not found.
func (c *Discord) pinnedStatusMessage(chanID, header string) (*discordgo.Message, error) {
	ms, err := c.s.ChannelMessagesPinned(chanID)
	if err != nil {
		return nil, err
	}
	var statusMessage *discordgo.Message
	for _, m := range ms {
		if len(m.Embeds) > 0 && m.Embeds[0].Author.Name == header {
			if statusMessage != nil {
				log.Printf("Multiple status messages in %v, editing last one", chanID)
			}
			statusMessage = m
		}
	}
	return statusMessage, nil
}

// Set the pinned status message, by posting one or editing the existing one.
// No-op if the status was already set.
func (c *Discord) CreateUpdatePin(chanID, header string, embed *discordgo.MessageEmbed) error {
	statusMessage, err := c.pinnedStatusMessage(chanID, header)
	if err != nil {
		return err
	}

	if statusMessage == nil {
		m, err := c.s.ChannelMessageSendEmbed(chanID, embed)
		if err != nil {
			return err
		}
		return c.s.ChannelMessagePin(chanID, m.ID)
	} else if statusMessage.Embeds[0] == embed {
		return nil // no-op
	}

	_, err = c.s.ChannelMessageEditEmbed(chanID, statusMessage.ID, embed)
	return err
}

func (c *Discord) QMChannelSend(msg string) error {
	_, err := c.s.ChannelMessageSend(c.QMChannelID, msg)
	return err
}

func (c *Discord) GeneralChannelSend(msg string) error {
	_, err := c.s.ChannelMessageSend(c.generalChannelID, msg)
	return err
}

func (c *Discord) GeneralChannelSendEmbed(embed *discordgo.MessageEmbed) error {
	_, err := c.s.ChannelMessageSendEmbed(c.generalChannelID, embed)
	return err
}

func (c *Discord) StatusUpdateChannelSend(msg string) error {
	_, err := c.s.ChannelMessageSend(c.statusUpdateChannelID, msg)
	return err
}

func (c *Discord) TechChannelSend(msg string) error {
	_, err := c.s.ChannelMessageSend(c.techChannelID, msg)
	return err
}

var ErrChannelNotFound = fmt.Errorf("channel not found")

// Returns whether a channel was archived.
func (c *Discord) ArchiveChannel(chID string) error {
	ch, err := c.s.Channel(chID)
	if err != nil {
		return fmt.Errorf("channel id %s not found: %w", chID, ErrChannelNotFound)
	}

	// Already archived.
	if ch.ParentID != "" {
		parentCh, err := c.s.Channel(ch.ParentID)
		if err != nil {
			return fmt.Errorf("parent channel id %s not found: %w", ch.ParentID, ErrChannelNotFound)
		}
		if strings.HasPrefix(parentCh.Name, "Solved") {
			return nil
		}
	}

	arCh, err := c.s.Channel(c.solvedCategoryID)
	if err != nil {
		return fmt.Errorf("error looking up archive: %v", err)
	}

	_, err = c.s.ChannelEditComplex(chID, &discordgo.ChannelEdit{ParentID: c.solvedCategoryID, PermissionOverwrites: arCh.PermissionOverwrites})
	if err != nil {
		return fmt.Errorf("error moving channel: %v", err)
	}
	return nil
}

// CreateChannel ensures that a channel exists with the given name, and returns the channel ID.
func (c *Discord) CreateChannel(name string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if id, ok := c.channelNameToID[name]; ok {
		return id, nil
	}
	ch, err := c.s.GuildChannelCreateComplex(c.GuildID, discordgo.GuildChannelCreateData{
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

func (c *Discord) ClosestRoomID(input string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for r, id := range c.roomsToID {
		input = strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(input), " ", ""), "-", "")
		r = strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(r), " ", ""), "-", "")
		if r == input {
			return id, true
		}
	}
	return "", false
}

func (c *Discord) AvailableRooms() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var rs []string
	for r := range c.roomsToID {
		rs = append(rs, r)
	}
	return rs
}

type voiceRoom struct {
	// The name of the room, eg. "Patio". This excludes the puzzles that might be part of the channel name.
	name    string
	puzzles []string
}

func parseRoom(voiceChanName string) (voiceRoom, error) {
	parts := strings.Split(voiceChanName, ":")
	if len(parts) == 1 {
		return voiceRoom{name: parts[0]}, nil
	}
	if len(parts) != 2 {
		return voiceRoom{}, fmt.Errorf("too many ':' in voice channel name: %q", voiceChanName)
	}
	puzzles := strings.Split(parts[1], ",")
	for i, p := range puzzles {
		puzzles[i] = strings.TrimSpace(p)
	}
	return voiceRoom{
		name:    parts[0],
		puzzles: puzzles,
	}, nil
}
