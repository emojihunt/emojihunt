package db

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/schema"
	"golang.org/x/xerrors"
)

func (c *Client) SetDiscordChannel(ctx context.Context, puzzle *schema.Puzzle,
	channel string) (*schema.Puzzle, error) {

	result, err := c.queries.UpdateDiscordChannel(ctx, UpdateDiscordChannelParams{
		ID: puzzle.ID, DiscordChannel: channel,
	})
	if err != nil {
		return nil, xerrors.Errorf("SetDiscordChannel: %w", err)
	}
	return c.parseDatabaseResult(&result), nil
}

func (c *Client) SetSpreadsheetID(ctx context.Context, puzzle *schema.Puzzle,
	spreadsheet string) (*schema.Puzzle, error) {

	result, err := c.queries.UpdateSpreadsheetID(ctx, UpdateSpreadsheetIDParams{
		ID: puzzle.ID, SpreadsheetID: spreadsheet,
	})
	if err != nil {
		return nil, xerrors.Errorf("UpdateSpreadsheetID: %w", err)
	}
	return c.parseDatabaseResult(&result), nil
}

func (c *Client) SetStatusAndAnswer(ctx context.Context, puzzle *schema.Puzzle,
	status schema.Status, answer string) (*schema.Puzzle, error) {

	result, err := c.queries.UpdateStatusAndAnswer(ctx, UpdateStatusAndAnswerParams{
		ID: puzzle.ID, Status: "TODO", Answer: answer, Archived: status.IsSolved(),
	})
	if err != nil {
		return nil, xerrors.Errorf("UpdateStatusAndAnswer: %w", err)
	}
	return c.parseDatabaseResult(&result), nil
}

func (c *Client) SetDescription(ctx context.Context, puzzle *schema.Puzzle,
	description string) (*schema.Puzzle, error) {

	result, err := c.queries.UpdateDescription(ctx, UpdateDescriptionParams{
		ID: puzzle.ID, Description: description,
	})
	if err != nil {
		return nil, xerrors.Errorf("UpdateDescription: %w", err)
	}
	return c.parseDatabaseResult(&result), nil
}

func (c *Client) SetLocation(ctx context.Context, puzzle *schema.Puzzle,
	location string) (*schema.Puzzle, error) {

	result, err := c.queries.UpdateLocation(ctx, UpdateLocationParams{
		ID: puzzle.ID, Location: location,
	})
	if err != nil {
		return nil, xerrors.Errorf("UpdateLocation: %w", err)
	}
	return c.parseDatabaseResult(&result), nil
}

func (c *Client) SetBotFields(ctx context.Context, puzzle *schema.Puzzle) (
	*schema.Puzzle, error) {

	result, err := c.queries.UpdateArchived(ctx, UpdateArchivedParams{
		ID: puzzle.ID, Archived: puzzle.ShouldArchive(),
	})
	if err != nil {
		return nil, xerrors.Errorf("UpdateArchived: %w", err)
	}
	return c.parseDatabaseResult(&result), nil
}

func (c *Client) SetVoiceRoom(ctx context.Context, puzzle *schema.Puzzle,
	channel *discordgo.Channel) (*schema.Puzzle, error) {

	var channelID, channelName string
	if channel != nil {
		channelID = channel.ID
		channelName = channel.Name
	}
	result, err := c.queries.UpdateVoiceRoom(ctx, UpdateVoiceRoomParams{
		ID: puzzle.ID, VoiceRoom: channelID, Location: channelName,
	})
	if err != nil {
		return nil, xerrors.Errorf("UpdateVoiceRoom: %w", err)
	}
	return c.parseDatabaseResult(&result), nil
}
