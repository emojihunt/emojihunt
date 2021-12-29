package syncer

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/schema"
)

func (s *Syncer) MarkSolved(ctx context.Context, puzzle *schema.Puzzle) error {
	verb := "solved"
	if puzzle.Status == schema.Backsolved {
		verb = "backsolved"
	}

	if puzzle.Answer == "" {
		if err := s.discord.ChannelSend(puzzle.DiscordChannel, fmt.Sprintf("Puzzle %s!  Please add the answer to the sheet.", verb)); err != nil {
			return fmt.Errorf("error posting solved puzzle announcement: %v", err)
		}

		if err := s.discord.QMChannelSend(fmt.Sprintf("Puzzle %q marked %s, but has no answer, please add it to the sheet.", puzzle.Name, verb)); err != nil {
			return fmt.Errorf("error posting solved puzzle announcement: %v", err)
		}

		return nil // don't archive until we have the answer.
	}

	archived, err := s.discord.ArchiveChannel(puzzle.DiscordChannel)
	if !archived {
		// Channel already archived (cache is best-effort -- this can happen
		// after restart or if a human did it)
	} else if err != nil {
		return fmt.Errorf("unable to archive channel for %q: %v", puzzle.Name, err)
	} else {
		log.Printf("Archiving channel for %q", puzzle.Name)
		// post to relevant channels only if it was newly archived.
		if err := s.discord.ChannelSend(puzzle.DiscordChannel, fmt.Sprintf("Puzzle %s! The answer was `%v`. I'll archive this channel.", verb, puzzle.Answer)); err != nil {
			return fmt.Errorf("error posting solved puzzle announcement: %v", err)
		}

		embed := &discordgo.MessageEmbed{
			Author: &discordgo.MessageEmbedAuthor{
				Name:    fmt.Sprintf("Puzzle %s!", verb),
				IconURL: puzzle.Round.TwemojiURL(),
			},
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Channel",
					Value:  fmt.Sprintf("<#%s>", puzzle.DiscordChannel),
					Inline: true,
				},
				{
					Name:   "Answer",
					Value:  fmt.Sprintf("`%s`", puzzle.Answer),
					Inline: true,
				},
			},
		}

		if err := s.discord.GeneralChannelSendEmbed(embed); err != nil {
			return fmt.Errorf("error posting solved puzzle announcement: %v", err)
		}
	}

	log.Printf("Marking sheet solved for %q", puzzle.Name)
	return s.drive.MarkSheetSolved(ctx, puzzle.SpreadsheetID)
}
