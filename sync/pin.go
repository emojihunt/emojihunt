package sync

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/state/status"
)

type DiscordPinFields struct {
	RoundName  string
	RoundEmoji string

	PuzzleName     string
	Status         status.Status
	Note           string
	Location       string
	PuzzleURL      string
	SpreadsheetID  string
	DiscordChannel string
	VoiceRoom      string
}

func NewDiscordPinFields(puzzle state.Puzzle) DiscordPinFields {
	return DiscordPinFields{
		RoundName:      puzzle.Round.Name,
		RoundEmoji:     puzzle.Round.Emoji,
		PuzzleName:     puzzle.Name,
		Status:         puzzle.Status,
		Note:           puzzle.Note,
		Location:       puzzle.Location,
		PuzzleURL:      puzzle.PuzzleURL,
		SpreadsheetID:  puzzle.SpreadsheetID,
		DiscordChannel: puzzle.DiscordChannel,
		VoiceRoom:      puzzle.VoiceRoom,
	}
}

// UpdateDiscordPin creates or updates the pinned message at the top of the
// puzzle channel. This message contains information about the puzzle status as
// well as links to the puzzle and the spreadsheet.
func (c *Client) UpdateDiscordPin(fields DiscordPinFields) error {
	log.Printf("sync: updating discord pin for %q", fields.PuzzleName)

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{Name: pinnedStatusHeader},
		Title:  fields.PuzzleName,
		URL:    fields.PuzzleURL,
		Color:  embedColor,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Round",
				Value:  fmt.Sprintf("%s %s", fields.RoundEmoji, fields.RoundName),
				Inline: false,
			},
			{
				Name:   "Status",
				Value:  fields.Status.Pretty(),
				Inline: true,
			},
			{
				Name:   "Puzzle",
				Value:  fmt.Sprintf("[Link](%s)", fields.PuzzleURL),
				Inline: true,
			},
		},
	}

	if fields.SpreadsheetID != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name: "Sheet",
			Value: fmt.Sprintf("[Link](https://docs.google.com/spreadsheets/d/%s)",
				fields.SpreadsheetID),
			Inline: true,
		})
	}

	if fields.Note != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Note",
			Value:  fields.Note,
			Inline: false,
		})
	}

	if !fields.Status.IsSolved() {
		locationMsg := locationDefaultMsg
		if fields.VoiceRoom != "" {
			locationMsg = fmt.Sprintf("Join us in <#%s>!", fields.VoiceRoom)
		}
		if fields.Location != "" {
			if locationMsg != locationDefaultMsg {
				locationMsg += fmt.Sprintf("Also in-person in %s.", fields.Location)
			} else {
				locationMsg = fmt.Sprintf("In-person in %s.", fields.Location)
			}
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Location",
			Value:  locationMsg,
			Inline: false,
		})
	}

	return c.discord.CreateUpdatePin(fields.DiscordChannel, pinnedStatusHeader, embed)
}
