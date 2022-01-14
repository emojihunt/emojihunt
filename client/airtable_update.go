package client

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/schema"
)

// Set[...] updates the given fields in Airtable and returns the updated record.
// The caller *must* hold the puzzle lock. The unlock function is passed through
// to the updated puzzle object unchanged.

func (air *Airtable) SetDiscordChannel(puzzle *schema.Puzzle, channel string) (*schema.Puzzle, error) {
	record, err := puzzle.AirtableRecord.UpdateRecordPartial(map[string]interface{}{
		"Discord Channel": channel,
	})
	if err != nil {
		return nil, err
	}
	return air.parseRecord(record, puzzle.Unlock)
}

func (air *Airtable) SetSpreadsheetID(puzzle *schema.Puzzle, spreadsheet string) (*schema.Puzzle, error) {
	record, err := puzzle.AirtableRecord.UpdateRecordPartial(map[string]interface{}{
		"Spreadsheet ID": spreadsheet,
	})
	if err != nil {
		return nil, err
	}
	return air.parseRecord(record, puzzle.Unlock)
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
	return air.parseRecord(record, puzzle.Unlock)
}

func (air *Airtable) SetDescription(puzzle *schema.Puzzle, description string) (*schema.Puzzle, error) {
	var fields = map[string]interface{}{
		"Description": description,
	}
	record, err := puzzle.AirtableRecord.UpdateRecordPartial(fields)
	if err != nil {
		return nil, err
	}
	return air.parseRecord(record, puzzle.Unlock)
}

func (air *Airtable) SetNotes(puzzle *schema.Puzzle, notes string) (*schema.Puzzle, error) {
	var fields = map[string]interface{}{
		"Notes": notes,
	}
	record, err := puzzle.AirtableRecord.UpdateRecordPartial(fields)
	if err != nil {
		return nil, err
	}
	return air.parseRecord(record, puzzle.Unlock)
}

func (air *Airtable) SetBotFields(puzzle *schema.Puzzle, lastBotStatus schema.Status, archived, pending bool) (*schema.Puzzle, error) {
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
	return air.parseRecord(record, puzzle.Unlock)
}

func (air *Airtable) SetVoiceRoom(puzzle *schema.Puzzle, channel *discordgo.Channel) (*schema.Puzzle, error) {
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
	return air.parseRecord(record, puzzle.Unlock)
}
