package client

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/schema"
	"github.com/mehanizm/airtable"
)

func (air *Airtable) UpdateDiscordChannel(puzzle *schema.Puzzle, channel string) (*schema.Puzzle, error) {
	record, err := puzzle.AirtableRecord.UpdateRecordPartial(map[string]interface{}{
		"Discord Channel": channel,
	})
	if err != nil {
		return nil, err
	}
	return air.parseRecord(record)
}

func (air *Airtable) UpdateSpreadsheetID(puzzle *schema.Puzzle, spreadsheet string) (*schema.Puzzle, error) {
	record, err := puzzle.AirtableRecord.UpdateRecordPartial(map[string]interface{}{
		"Spreadsheet ID": spreadsheet,
	})
	if err != nil {
		return nil, err
	}
	return air.parseRecord(record)
}

func (air *Airtable) SetStatusAndAnswer(puzzle *schema.Puzzle, status schema.Status, answer string) (*schema.Puzzle, error) {
	var fields = map[string]interface{}{
		"Status":          status.PrettyForAirtable(),
		"Answer":          answer,
		"Last Bot Status": status.TextForAirtable(),
		"Last Bot Sync":   time.Now().Format(time.RFC3339),
		"Archived":        status.IsSolved(),
	}
	record, err := puzzle.AirtableRecord.UpdateRecordPartial(fields)
	if err != nil {
		return nil, err
	}
	return air.parseRecord(record)
}

func (air *Airtable) SetDescription(puzzle *schema.Puzzle, description string) (*schema.Puzzle, error) {
	var fields = map[string]interface{}{
		"Description": description,
	}
	record, err := puzzle.AirtableRecord.UpdateRecordPartial(fields)
	if err != nil {
		return nil, err
	}
	return air.parseRecord(record)
}

func (air *Airtable) SetNotes(puzzle *schema.Puzzle, notes string) (*schema.Puzzle, error) {
	var fields = map[string]interface{}{
		"Notes": notes,
	}
	record, err := puzzle.AirtableRecord.UpdateRecordPartial(fields)
	if err != nil {
		return nil, err
	}
	return air.parseRecord(record)
}

func (air *Airtable) UpdateBotFields(puzzle *schema.Puzzle, lastBotStatus schema.Status, archived, pending bool) (*schema.Puzzle, error) {
	var fields = make(map[string]interface{})

	if lastBotStatus == schema.NotStarted {
		fields["Last Bot Status"] = nil
	} else {
		fields["Last Bot Status"] = string(lastBotStatus)
	}

	fields["Archived"] = archived
	fields["Last Bot Sync"] = time.Now().Format(time.RFC3339)

	if puzzle.Pending != pending {
		// The "pending" status is stored in the puzzle name
		puzzleName := puzzle.Name
		if pending {
			puzzle.Name += pendingSuffix
		}
		fields["Name"] = puzzleName
	}

	record, err := puzzle.AirtableRecord.UpdateRecordPartial(fields)
	if err != nil {
		return nil, err
	}
	return air.parseRecord(record)
}

func (air *Airtable) UpdateVoiceRoom(puzzle *schema.Puzzle, channel *discordgo.Channel) (*schema.Puzzle, error) {
	var channelID string
	if channel != nil {
		channelID = channel.ID
	}
	record, err := puzzle.AirtableRecord.UpdateRecordPartial(map[string]interface{}{
		"Voice Room": channelID,
	})
	if err != nil {
		return nil, err
	}
	return air.parseRecord(record)
}

func (air *Airtable) AddPuzzles(puzzles []*schema.NewPuzzle) ([]*schema.Puzzle, error) {
	var created []*schema.Puzzle
	for i := 0; i < len(puzzles); i += 10 {
		records := airtable.Records{}
		limit := i + 10
		if limit > len(puzzles) {
			limit = len(puzzles)
		}
		for _, puzzle := range puzzles[i:limit] {
			fields := map[string]interface{}{
				"Name":         puzzle.Name + pendingSuffix,
				"Round":        puzzle.Round.Serialize(),
				"Puzzle URL":   puzzle.PuzzleURL,
				"Original URL": puzzle.PuzzleURL,
			}
			records.Records = append(records.Records,
				&airtable.Record{
					Fields: fields,
				},
			)
		}
		response, err := air.table.AddRecords(&records)
		if err != nil {
			return nil, err
		}
		for _, record := range response.Records {
			parsed, err := air.parseRecord(record)
			if err != nil {
				return nil, err
			}
			created = append(created, parsed)
		}
	}
	return created, nil
}
