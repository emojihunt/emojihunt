package client

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/db"
	"github.com/emojihunt/emojihunt/schema"
)

// Set[...] updates the given fields in Airtable and returns the updated record.
// The caller *must* hold the puzzle lock. The unlock function is passed through
// to the updated puzzle object unchanged.

func (air *Airtable) SetDiscordChannel(puzzle *schema.Puzzle, channel string) (*schema.Puzzle, error) {
	result, err := air.database.UpdateDiscordChannel(context.TODO(), db.UpdateDiscordChannelParams{
		ID: puzzle.AirtableRecord.ID, DiscordChannel: channel,
	})
	if err != nil {
		return nil, err
	}
	return air.parseDatabaseResult(&result, puzzle.Unlock), nil
}

func (air *Airtable) SetSpreadsheetID(puzzle *schema.Puzzle, spreadsheet string) (*schema.Puzzle, error) {
	result, err := air.database.UpdateSpreadsheetID(context.TODO(), db.UpdateSpreadsheetIDParams{
		ID: puzzle.AirtableRecord.ID, SpreadsheetID: spreadsheet,
	})
	if err != nil {
		return nil, err
	}
	return air.parseDatabaseResult(&result, puzzle.Unlock), nil
}

func (air *Airtable) SetStatusAndAnswer(puzzle *schema.Puzzle, status schema.Status, answer string) (*schema.Puzzle, error) {
	result, err := air.database.UpdateStatusAndAnswer(context.TODO(), db.UpdateStatusAndAnswerParams{
		ID: puzzle.AirtableRecord.ID, Status: string(status), Answer: answer, Archived: status.IsSolved(),
	})
	if err != nil {
		return nil, err
	}
	return air.parseDatabaseResult(&result, puzzle.Unlock), nil
}

func (air *Airtable) SetDescription(puzzle *schema.Puzzle, description string) (*schema.Puzzle, error) {
	result, err := air.database.UpdateDescription(context.TODO(), db.UpdateDescriptionParams{
		ID: puzzle.AirtableRecord.ID, Description: description,
	})
	if err != nil {
		return nil, err
	}
	return air.parseDatabaseResult(&result, puzzle.Unlock), nil
}

func (air *Airtable) SetLocation(puzzle *schema.Puzzle, location string) (*schema.Puzzle, error) {
	result, err := air.database.UpdateLocation(context.TODO(), db.UpdateLocationParams{
		ID: puzzle.AirtableRecord.ID, Location: location,
	})
	if err != nil {
		return nil, err
	}
	return air.parseDatabaseResult(&result, puzzle.Unlock), nil
}

func (air *Airtable) SetBotFields(puzzle *schema.Puzzle) (*schema.Puzzle, error) {
	result, err := air.database.UpdateArchived(context.TODO(), db.UpdateArchivedParams{
		ID: puzzle.AirtableRecord.ID, Archived: puzzle.ShouldArchive(),
	})
	if err != nil {
		return nil, err
	}
	return air.parseDatabaseResult(&result, puzzle.Unlock), nil
}

func (air *Airtable) SetVoiceRoom(puzzle *schema.Puzzle, channel *discordgo.Channel) (*schema.Puzzle, error) {
	var channelID, channelName string
	if channel != nil {
		channelID = channel.ID
		channelName = channel.Name
	}
	result, err := air.database.UpdateVoiceRoom(context.TODO(), db.UpdateVoiceRoomParams{
		ID: puzzle.AirtableRecord.ID, VoiceRoom: channelID, Location: channelName,
	})
	if err != nil {
		return nil, err
	}
	return air.parseDatabaseResult(&result, puzzle.Unlock), nil
}
