package syncer

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/schema"
)

func (s *Syncer) IdempotentUpdate(ctx context.Context, puzzle *schema.Puzzle) (*schema.Puzzle, error) {
	if puzzle.Status == puzzle.LastBotStatus {
		// Nothing to do!
		return puzzle, nil
	}

	if puzzle.Status.IsSolved() {
		if err := s.markSolved(ctx, puzzle); err != nil {
			return nil, fmt.Errorf("failed to mark puzzle %q solved: %v", puzzle.Name, err)
		}
	} else {
		if err := s.DiscordCreateUpdatePin(puzzle); err != nil {
			return nil, fmt.Errorf("unable to set puzzle status message for %q: %w", puzzle.Name, err)
		}
		if err := s.discord.StatusUpdateChannelSend(fmt.Sprintf("%s Puzzle <#%s> is now %v.", puzzle.Round.Emoji, puzzle.DiscordChannel, puzzle.Status.Pretty())); err != nil {
			return nil, fmt.Errorf("error posting puzzle status announcement: %v", err)
		}
	}

	// TODO: correctly trigger archiving (and unarchiving?) the Discord channel,
	// e.g. when the answer is added later

	var err error
	puzzle, err = s.airtable.UpdateLastBotStatus(puzzle, puzzle.Status)
	if err != nil {
		return nil, fmt.Errorf("failed to update Last Bot Status for puzzle %q: %v", puzzle.Name, err)
	}

	return puzzle, nil
}

func (s *Syncer) markSolved(ctx context.Context, puzzle *schema.Puzzle) error {
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

	log.Printf("Archiving channel for %q", puzzle.Name)
	if err := s.discord.ArchiveChannel(puzzle.DiscordChannel); err != nil {
		return fmt.Errorf("unable to archive channel for %q: %v", puzzle.Name, err)
	}

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

	log.Printf("Marking sheet solved for %q", puzzle.Name)
	return s.drive.MarkSheetSolved(ctx, puzzle.SpreadsheetID)
}
