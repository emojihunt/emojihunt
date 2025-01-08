package sync

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/state/status"
	"github.com/mazznoer/csscolorparser"
	"golang.org/x/xerrors"
)

const ServerURL = "https://www.emojihunt.org"

type DiscordPinFields struct {
	RoundName  string
	RoundEmoji string
	RoundHue   int64

	PuzzleID       int64
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
		RoundHue:       puzzle.Round.Hue,
		PuzzleID:       puzzle.ID,
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
func (c *Client) UpdateDiscordPin(ctx context.Context, fields DiscordPinFields) error {
	log.Printf("sync: updating discord pin for %q", fields.PuzzleName)
	css, err := csscolorparser.Parse(
		// csscolorparser makes poor choices when given colors outside of sRGB. The
		// lightness and chroma below were chosen so that all hues are within sRGB.
		fmt.Sprintf("oklch(65%% 0.11, %ddeg)", fields.RoundHue),
	)
	if err != nil {
		return xerrors.Errorf("csscolorparser: %w", err)
	}
	r, g, b, _ := css.RGBA255()
	color := int(r)*256*256 + int(g)*256 + int(b)

	embed := &discordgo.MessageEmbed{
		Title: fields.PuzzleName,
		URL:   fields.PuzzleURL,
		Color: color,
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
				Name:   "Links",
				Value:  fmt.Sprintf("[Puzzle](%s)", fields.PuzzleURL),
				Inline: true,
			},
		},
	}

	if fields.SpreadsheetID != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name: "—",
			Value: fmt.Sprintf(
				"[Sheet](%s/%d)  ·  [Backup](https://docs.google.com/spreadsheets/d/%s)",
				ServerURL, fields.PuzzleID, fields.SpreadsheetID,
			),
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
				locationMsg += fmt.Sprintf(" Also in-person in %s.", fields.Location)
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

	return c.discord.CreateUpdatePin(fields.DiscordChannel, embed)
}
