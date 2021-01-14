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
		return nil, fmt.Errorf("puzzle category %q not found", c.PuzzleCategoryName)
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
	roles, err := s.GuildRoles(guildID)
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
		guildID:               guildID,
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

const statusPrefix = "**=== Puzzle Information ===**"

// Returns last pinned status message, or nil if not found.
func (c *Client) pinnedStatusMessage(chanID string) (*discordgo.Message, error) {
	ms, err := c.s.ChannelMessagesPinned(chanID)
	if err != nil {
		return nil, err
	}
	var statusMessage *discordgo.Message
	for _, m := range ms {
		if strings.HasPrefix(m.Content, statusPrefix) {
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
func (c *Client) SetPinnedInfo(chanID, spreadsheetURL, puzzleURL, status string) (didUpdate bool, err error) {
	statusMessage, err := c.pinnedStatusMessage(chanID)
	if err != nil {
		return false, err
	}

	// TODO: embed
	msg := fmt.Sprintf("%s\nSpreadsheet: <%s>\nPuzzle: <%s>",
		statusPrefix, spreadsheetURL, puzzleURL)
	if status != "" {
		msg = fmt.Sprintf("%s\nStatus: %s", msg, status)
	}

	if statusMessage == nil {
		err := c.ChannelSendAndPin(chanID, msg)
		return err == nil, err
	} else if statusMessage.Content == msg {
		return false, nil // no-op
	}

	_, err = c.s.ChannelMessageEdit(chanID, statusMessage.ID, msg)
	return err == nil, err
}

func (c *Client) puzzleNameFromChannel(chanID string) (string, error) {
	m, err := c.pinnedStatusMessage(chanID)
	if err != nil {
		return "", fmt.Errorf("error getting pinned status message for channel ID %q: %v", chanID, err)
	}
	if m == nil {
		return "", fmt.Errorf("no status message for channel ID %q", chanID)
	}
	return "TODO puzzle name", nil
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
	if ch.ParentID == c.solvedCategoryID {
		return false, nil
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

func (c *Client) QMHandler(s *discordgo.Session, m *discordgo.MessageCreate) error {
	if m.Author.ID == s.State.User.ID || !strings.HasPrefix(m.Content, "!qm") || m.ChannelID != c.qmChannelID {
		return nil
	}

	var reply string
	var err error
	switch m.Content {
	case "!qm start":
		if err = s.GuildMemberRoleAdd(c.guildID, m.Author.ID, c.qmRoleID); err != nil {
			reply = fmt.Sprintf("unable to make %s a QM: %v", m.Author.Mention(), err)
			break
		}
		reply = fmt.Sprintf("%s is now a QM", m.Author.Mention())
	case "!qm stop":
		if err = s.GuildMemberRoleRemove(c.guildID, m.Author.ID, c.qmRoleID); err != nil {
			reply = fmt.Sprintf("unable to remove %s from QM role: %v", m.Author.Mention(), err)
			break
		}
		reply = fmt.Sprintf("%s is no longer a QM", m.Author.Mention())
	default:
		err = fmt.Errorf("unexpected QM command: %q", m.Content)
		reply = fmt.Sprintf("unexpected command: %q\nsupported qm commands are \"!qm start\" and \"!qm stop\"", m.Content)
	}
	if err != nil {
		log.Printf("error setting QM: %v", err)
	}
	_, err = s.ChannelMessageSend(m.ChannelID, reply)
	return err
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
	defer c.mu.Unlock()
	_, err := c.s.ChannelEdit(c.roomsToID[r.name], r.VoiceChannelName())
	return err
}

// Returns whether a puzzle was added.
func (c *Client) AddPuzzleToRoom(puzzle, roomID string) (bool, error) {
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
	roomCh, err := c.s.Channel(roomID)
	if err != nil {
		return false, fmt.Errorf("error finding room ID %q: %v", roomID, err)
	}
	r, err := parseRoom(roomCh.Name)
	if err != nil {
		return false, fmt.Errorf("error parsing room when removing puzzles: %v", err)
	}
	index := -1
	for i, p := range r.puzzles {
		if p == puzzle {
			index = i
		}
	}
	if index == -1 {
		// puzzle is not in this voice channel.
		return false, nil
	}
	r.puzzles = append(r.puzzles[:index], r.puzzles[index+1:]...)
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
