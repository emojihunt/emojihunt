package discord

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

type Config struct {
	GuildID                                                                     string
	QMChannelName, GeneralChannelName, StatusUpdateChannelName, TechChannelName string
	PuzzleCategoryName, SolvedCategoryName                                      string
	QMRoleName                                                                  string
}

type Client struct {
	s       *discordgo.Session
	guildID string
	// The QM channel contains a central log of interesting bot actions, as well as the only place for
	// advanced bot usage, such as puzzle or round creation.
	qmChannelID string
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
	qmRoleID string

	// This might be a case where sync.Map makes sense.
	mu              sync.Mutex
	channelNameToID map[string]string
	// A room is the common name of a voice channel, excluding the puzzles present there.
	// For example, a voice channel "Patio: puzzle 1, puzzle 2", is named "Patio", and has 2 puzzles being worked on.
	roomsToID map[string]string
}

func getGuildID(s *discordgo.Session) (string, error) {
	log.Print("fetching GuildID from session")
	gs := s.State.Guilds
	if len(gs) != 1 {
		return "", fmt.Errorf("expected exactly 1 guild, found %d", len(gs))
	}
	return gs[0].ID, nil
}

func New(s *discordgo.Session, c Config) (*Client, error) {
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

	return &Client{
		s:                     s,
		guildID:               c.GuildID,
		qmChannelID:           qm,
		generalChannelID:      gen,
		statusUpdateChannelID: st,
		techChannelID:         tech,
		channelNameToID:       chIDs,
		roomsToID:             rIDs,
		puzzleCategoryID:      puz,
		solvedCategoryID:      ar,
		qmRoleID:              qmRoleID,
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

func (c *Client) ChannelSendEmbedAndPin(chanID string, embed *discordgo.MessageEmbed) error {
	m, err := c.s.ChannelMessageSendEmbed(chanID, embed)
	if err != nil {
		return err
	}
	err = c.s.ChannelMessagePin(chanID, m.ID)
	return err
}

const statusTitle = "Puzzle Information"

// Returns last pinned status message, or nil if not found.
func (c *Client) pinnedStatusMessage(chanID, header string) (*discordgo.Message, error) {
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

type statusMessage struct {
	spreadsheetURL, puzzleURL string
	status                    string
}

// Set the pinned status message, by posting one or editing the existing one.
// No-op if the status was already set.
func (c *Client) CreateUpdatePin(chanID, header string, embed *discordgo.MessageEmbed) (didUpdate bool, err error) {
	statusMessage, err := c.pinnedStatusMessage(chanID, header)
	if err != nil {
		return false, err
	}

	if statusMessage == nil {
		err := c.ChannelSendEmbedAndPin(chanID, embed)
		return err == nil, err
	} else if statusMessage.Embeds[0] == embed {
		return false, nil // no-op
	}

	_, err = c.s.ChannelMessageEditEmbed(chanID, statusMessage.ID, embed)
	return err == nil, err
}

func (c *Client) QMChannelSend(msg string) error {
	_, err := c.s.ChannelMessageSend(c.qmChannelID, msg)
	return err
}

func (c *Client) GeneralChannelSend(msg string) error {
	_, err := c.s.ChannelMessageSend(c.generalChannelID, msg)
	return err
}

func (c *Client) GeneralChannelSendEmbed(embed *discordgo.MessageEmbed) error {
	_, err := c.s.ChannelMessageSendEmbed(c.generalChannelID, embed)
	return err
}

func (c *Client) StatusUpdateChannelSend(msg string) error {
	_, err := c.s.ChannelMessageSend(c.statusUpdateChannelID, msg)
	return err
}

func (c *Client) TechChannelSend(msg string) error {
	_, err := c.s.ChannelMessageSend(c.techChannelID, msg)
	return err
}

var ChannelNotFound = fmt.Errorf("channel not found")

// Returns whether a channel was archived.
func (c *Client) ArchiveChannel(chID string) (bool, error) {
	ch, err := c.s.Channel(chID)
	if err != nil {
		return false, fmt.Errorf("channel id %s not found: %w", chID, ChannelNotFound)
	}

	// Already archived.
	if ch.ParentID != "" {
		parentCh, err := c.s.Channel(ch.ParentID)
		if err != nil {
			return false, fmt.Errorf("parent channel id %s not found: %w", ch.ParentID, ChannelNotFound)
		}
		if strings.HasPrefix(parentCh.Name, "Solved") {
			return false, nil
		}
	}

	arCh, err := c.s.Channel(c.solvedCategoryID)
	if err != nil {
		return false, fmt.Errorf("error looking up archive: %v", err)
	}

	_, err = c.s.ChannelEditComplex(chID, &discordgo.ChannelEdit{ParentID: c.solvedCategoryID, PermissionOverwrites: arCh.PermissionOverwrites})
	if err != nil {
		return false, fmt.Errorf("error moving channel: %v", err)
	}
	return true, nil
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

func (c *Client) ClosestRoomID(input string) (string, bool) {
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

func (c *Client) AvailableRooms() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var rs []string
	for r := range c.roomsToID {
		rs = append(rs, r)
	}
	return rs
}

func (c *Client) updateRoom(r room) error {
	c.mu.Lock()
	rID := c.roomsToID[r.name]
	c.mu.Unlock()
	log.Printf("channel update for room: %v ...", r)
	defer log.Printf("channel update for room: %v ... done", r)
	_, err := c.s.ChannelEdit(rID, r.VoiceChannelName())
	return err
}

// Returns whether a puzzle was added.
func (c *Client) AddPuzzleToRoom(puzzle, roomID string) (bool, error) {
	log.Printf("adding puzzle %q to room %q ...", puzzle, roomID)
	defer log.Printf("adding puzzle %q to room %q ... done", puzzle, roomID)
	roomCh, err := c.s.Channel(roomID)
	if err != nil {
		return false, fmt.Errorf("error finding room ID %q: %v", roomID, err)
	}
	r, err := parseRoom(roomCh.Name)
	if err != nil {
		return false, fmt.Errorf("error parsing room when adding puzzles: %v", err)
	}
	for _, p := range r.puzzles {
		if p == puzzle {
			return false, nil
		}
	}
	r.puzzles = append(r.puzzles, puzzle)
	if err := c.updateRoom(r); err != nil {
		return false, err
	}
	return true, nil
}

// Returns whether a puzzle was removed.
func (c *Client) RemovePuzzleFromRoom(puzzle, roomID string) (bool, error) {
	log.Printf("removing %q from %q ...", puzzle, roomID)
	defer log.Printf("removing %q from %q ... done", puzzle, roomID)
	roomCh, err := c.s.Channel(roomID)
	if err != nil {
		return false, fmt.Errorf("error finding room ID %q: %v", roomID, err)
	}
	r, err := parseRoom(roomCh.Name)
	if err != nil {
		return false, fmt.Errorf("error parsing room when removing puzzles: %v", err)
	}
	log.Printf("parsed room as %v", r)
	index := -1
	for i, p := range r.puzzles {
		if p == puzzle {
			index = i
		}
	}
	if index == -1 {
		// puzzle is not in this voice channel.
		log.Printf("REMOVE NOISY LOG: did not find puzzle %q in channelID %q", puzzle, roomID)
		return false, nil
	}
	r.puzzles = append(r.puzzles[:index], r.puzzles[index+1:]...)
	log.Printf("updated room: %v", r)
	if err := c.updateRoom(r); err != nil {
		return false, err
	}
	return true, nil
}

var voiceRE = regexp.MustCompile(`!voice (start|stop) (.*)$`)

type room struct {
	// The name of the room, eg. "Patio". This excludes the puzzles that might be part of the channel name.
	name    string
	puzzles []string
}

func (r room) VoiceChannelName() string {
	if len(r.puzzles) == 0 {
		return r.name
	}
	return fmt.Sprintf("%s: %s", r.name, strings.Join(r.puzzles, ", "))
}

func parseRoom(voiceChanName string) (room, error) {
	parts := strings.Split(voiceChanName, ":")
	if len(parts) == 1 {
		return room{name: parts[0]}, nil
	}
	if len(parts) != 2 {
		return room{}, fmt.Errorf("too many ':' in voice channel name: %q", voiceChanName)
	}
	puzzles := strings.Split(parts[1], ",")
	for i, p := range puzzles {
		puzzles[i] = strings.TrimSpace(p)
	}
	return room{
		name:    parts[0],
		puzzles: puzzles,
	}, nil
}

func ChannelMention(chanID string) string {
	return fmt.Sprintf("<#%s>", chanID)
}
