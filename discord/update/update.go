package update

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type Updater struct {
	s               *discordgo.Session
	guildID         string
	qmChannel       string
	qmChannelID     string
	channelNameToID map[string]string
}

func getGuildID(s *discordgo.Session) (string, error) {
	gs := s.State.Guilds
	if len(gs) != 1 {
		return "", fmt.Errorf("expected exactly 1 guild, found %d", len(gs))
	}
	return gs[0].ID, nil
}

func New(s *discordgo.Session, qmChannel string) (*Updater, error) {
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
	qmID, ok := chIDs[qmChannel]
	if !ok {
		return nil, fmt.Errorf("QM Channel %q not found", qmChannel)
	}

	return &Updater{
		s:               s,
		guildID:         guildID,
		qmChannel:       qmChannel,
		qmChannelID:     qmID,
		channelNameToID: chIDs,
	}, nil
}

func (u *Updater) QMChannelSend(msg string) error {
	_, err := u.s.ChannelMessageSend(u.qmChannelID, "hello")
	return err
}

func (u *Updater) SolvePuzzle(puzzleName string) error {
	return u.QMChannelSend("hello")
}
