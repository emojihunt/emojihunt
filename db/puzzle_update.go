package db

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/schema"
)

// Set[...] updates the given fields in Airtable and returns the updated record.
// The caller *must* hold the puzzle lock. The unlock function is passed through
// to the updated puzzle object unchanged.

func (c *Client) SetDiscordChannel(puzzle *schema.Puzzle, channel string) (*schema.Puzzle, error) {
	result, err := c.queries.UpdateDiscordChannel(context.TODO(), UpdateDiscordChannelParams{
		ID: puzzle.ID, DiscordChannel: channel,
	})
	if err != nil {
		return nil, err
	}
	return c.parseDatabaseResult(&result, puzzle.Unlock), nil
}

func (c *Client) SetSpreadsheetID(puzzle *schema.Puzzle, spreadsheet string) (*schema.Puzzle, error) {
	result, err := c.queries.UpdateSpreadsheetID(context.TODO(), UpdateSpreadsheetIDParams{
		ID: puzzle.ID, SpreadsheetID: spreadsheet,
	})
	if err != nil {
		return nil, err
	}
	return c.parseDatabaseResult(&result, puzzle.Unlock), nil
}

func (c *Client) SetStatusAndAnswer(puzzle *schema.Puzzle, status schema.Status, answer string) (*schema.Puzzle, error) {
	result, err := c.queries.UpdateStatusAndAnswer(context.TODO(), UpdateStatusAndAnswerParams{
		ID: puzzle.ID, Status: status, Answer: answer, Archived: status.IsSolved(),
	})
	if err != nil {
		return nil, err
	}
	return c.parseDatabaseResult(&result, puzzle.Unlock), nil
}

func (c *Client) SetDescription(puzzle *schema.Puzzle, description string) (*schema.Puzzle, error) {
	result, err := c.queries.UpdateDescription(context.TODO(), UpdateDescriptionParams{
		ID: puzzle.ID, Description: description,
	})
	if err != nil {
		return nil, err
	}
	return c.parseDatabaseResult(&result, puzzle.Unlock), nil
}

func (c *Client) SetLocation(puzzle *schema.Puzzle, location string) (*schema.Puzzle, error) {
	result, err := c.queries.UpdateLocation(context.TODO(), UpdateLocationParams{
		ID: puzzle.ID, Location: location,
	})
	if err != nil {
		return nil, err
	}
	return c.parseDatabaseResult(&result, puzzle.Unlock), nil
}

func (c *Client) SetBotFields(puzzle *schema.Puzzle) (*schema.Puzzle, error) {
	result, err := c.queries.UpdateArchived(context.TODO(), UpdateArchivedParams{
		ID: puzzle.ID, Archived: puzzle.ShouldArchive(),
	})
	if err != nil {
		return nil, err
	}
	return c.parseDatabaseResult(&result, puzzle.Unlock), nil
}

func (c *Client) SetVoiceRoom(puzzle *schema.Puzzle, channel *discordgo.Channel) (*schema.Puzzle, error) {
	var channelID, channelName string
	if channel != nil {
		channelID = channel.ID
		channelName = channel.Name
	}
	result, err := c.queries.UpdateVoiceRoom(context.TODO(), UpdateVoiceRoomParams{
		ID: puzzle.ID, VoiceRoom: channelID, Location: channelName,
	})
	if err != nil {
		return nil, err
	}
	return c.parseDatabaseResult(&result, puzzle.Unlock), nil
}
