package client

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
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
	GeneralChannelID string
	// The channel in which to post status updates.
	StatusUpdateChannelID string
	// The tech channel has error messages.
	TechChannelID string
	// The puzzle channel category.
	PuzzleCategoryID string
	// The category for solved puzzles.
	SolvedCategoryID string
	// The Role ID for the QM role.
	QMRoleID string

	handlers map[string]DiscordCommandHandler

	// This might be a case where sync.Map makes sense.
	mu              sync.Mutex
	channelNameToID map[string]string
	roomsToID       map[string]string
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
		if ch.Type == discordgo.ChannelTypeGuildVoice {
			rIDs[ch.Name] = ch.ID
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

	commandHandlers := make(map[string]DiscordCommandHandler)
	discord := &Discord{
		s:                     s,
		GuildID:               c.GuildID,
		QMChannelID:           qm,
		GeneralChannelID:      gen,
		StatusUpdateChannelID: st,
		TechChannelID:         tech,
		channelNameToID:       chIDs,
		roomsToID:             rIDs,
		PuzzleCategoryID:      puz,
		SolvedCategoryID:      ar,
		QMRoleID:              qmRoleID,
		handlers:              commandHandlers,
	}
	s.AddHandler(discord.commandHandler)

	return discord, nil
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
	c.s.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if err := h(s, m); err != nil {
			log.Printf("%s: %v", name, err)
		}
	})
}

type DiscordCommand struct {
	ApplicationCommand *discordgo.ApplicationCommand
	Handler            DiscordCommandHandler
}

type DiscordCommandInput struct {
	IC         *discordgo.InteractionCreate
	User       *discordgo.User
	Command    string
	Subcommand string
}

type DiscordCommandHandler func(*discordgo.Session, *DiscordCommandInput) (string, error)

func (c *Discord) RegisterCommands(commands []*DiscordCommand) error {
	var appCommands []*discordgo.ApplicationCommand
	for _, command := range commands {
		appCommands = append(appCommands, command.ApplicationCommand)
		c.handlers[command.ApplicationCommand.Name] = command.Handler
	}
	_, err := c.s.ApplicationCommandBulkOverwrite(c.s.State.User.ID, c.GuildID, appCommands)
	return err
}

func (c *Discord) commandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	input := &DiscordCommandInput{
		IC:         i,
		User:       i.User,
		Command:    i.ApplicationCommandData().Name,
		Subcommand: "",
	}
	if input.User == nil {
		input.User = i.Member.User
	}
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Type == discordgo.ApplicationCommandOptionSubCommand {
			input.Subcommand = opt.Name
		}
	}

	if handler, ok := c.handlers[input.Command]; ok {
		log.Printf("discord: handling command %q from @%s",
			strings.Join([]string{input.Command, input.Subcommand}, " "), input.User.Username)

		// Call the handler! We need to run our logic and call
		// InteractionRespond within 3 seconds, otherwise Discord will report an
		// error to the user.
		reply, err := handler(s, input)
		if err != nil {
			log.Printf("discord: error handling interaction %q: %s", input.Command, spew.Sdump(err))
			reply = fmt.Sprintf("```\n🚨 Bot Error\n%s\n```", spew.Sdump(err))
		}

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: reply},
		})
		if err != nil {
			log.Printf("discord: error responding to interaction %q: %s", input.Command, spew.Sdump(err))
		}
	} else {
		log.Printf("discord: received unknown interaction: %#v %#v", i, i.ApplicationCommandData())
	}
}

func (c *Discord) ChannelSend(chanID, msg string) error {
	_, err := c.s.ChannelMessageSend(chanID, msg)
	return err
}

func (c *Discord) ChannelSendEmbed(chanID string, embed *discordgo.MessageEmbed) error {
	_, err := c.s.ChannelMessageSendEmbed(chanID, embed)
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

var ErrChannelNotFound = fmt.Errorf("channel not found")

func (c *Discord) SetChannelCategory(chID, categoryID string) error {
	ch, err := c.s.Channel(chID)
	if err != nil {
		return fmt.Errorf("channel id %s not found: %w", chID, ErrChannelNotFound)
	}

	if ch.ParentID == categoryID {
		return nil // no-op
	}

	category, err := c.s.Channel(categoryID)
	if err != nil {
		return fmt.Errorf("category channel id %s not found: %w", categoryID, ErrChannelNotFound)
	}

	_, err = c.s.ChannelEditComplex(chID, &discordgo.ChannelEdit{
		ParentID:             categoryID,
		PermissionOverwrites: category.PermissionOverwrites,
	})
	if err != nil {
		return fmt.Errorf("error moving channel: %v", err)
	}
	return nil
}

func (c *Discord) SetChannelName(chID, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.channelNameToID[name] = chID

	_, err := c.s.ChannelEdit(chID, name)
	return err
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
		ParentID: c.PuzzleCategoryID,
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
