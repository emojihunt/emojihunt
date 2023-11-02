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
	return nil, air.database.UpdateDiscordChannel(context.TODO(), db.UpdateDiscordChannelParams{
		ID: puzzle.AirtableRecord.ID, DiscordChannel: channel,
	}) // TODO: nil puzzle
}

func (air *Airtable) SetSpreadsheetID(puzzle *schema.Puzzle, spreadsheet string) (*schema.Puzzle, error) {
	return nil, air.database.UpdateSpreadsheetID(context.TODO(), db.UpdateSpreadsheetIDParams{
		ID: puzzle.AirtableRecord.ID, SpreadsheetID: spreadsheet,
	}) // TODO: nil puzzle
}

func (air *Airtable) SetStatusAndAnswer(puzzle *schema.Puzzle, status schema.Status, answer string) (*schema.Puzzle, error) {
	return nil, air.database.UpdateStatusAndAnswer(context.TODO(), db.UpdateStatusAndAnswerParams{
		ID: puzzle.AirtableRecord.ID, Status: string(status), Answer: answer, Archived: status.IsSolved(),
	}) // TODO: nil puzzle
}

func (air *Airtable) SetDescription(puzzle *schema.Puzzle, description string) (*schema.Puzzle, error) {
	return nil, air.database.UpdateDescription(context.TODO(), db.UpdateDescriptionParams{
		ID: puzzle.AirtableRecord.ID, Description: description,
	}) // TODO: nil puzzle
}

func (air *Airtable) SetLocation(puzzle *schema.Puzzle, location string) (*schema.Puzzle, error) {
	return nil, air.database.UpdateLocation(context.TODO(), db.UpdateLocationParams{
		ID: puzzle.AirtableRecord.ID, Location: location,
	}) // TODO: nil puzzle
}

func (air *Airtable) SetBotFields(puzzle *schema.Puzzle, lastBotStatus schema.Status, archived bool) (*schema.Puzzle, error) {
	return nil, air.database.UpdateArchived(context.TODO(), db.UpdateArchivedParams{
		ID: puzzle.AirtableRecord.ID, Archived: archived,
	}) // TODO: nil puzzle
}

func (air *Airtable) SetVoiceRoom(puzzle *schema.Puzzle, channel *discordgo.Channel) (*schema.Puzzle, error) {
	var channelID, channelName string
	if channel != nil {
		channelID = channel.ID
		channelName = channel.Name
	}
	return nil, air.database.UpdateVoiceRoom(context.TODO(), db.UpdateVoiceRoomParams{
		ID: puzzle.AirtableRecord.ID, VoiceRoom: channelID, Location: channelName,
	}) // TODO: nil puzzle
}
