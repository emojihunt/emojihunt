package db

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/xerrors"
)

func (c *Client) SetDiscordChannel(
	ctx context.Context, puzzle *Puzzle, channel string,
) (*Puzzle, error) {
	result, err := c.queries.UpdateDiscordChannel(ctx, UpdateDiscordChannelParams{
		ID: puzzle.ID, DiscordChannel: channel,
	})
	if err != nil {
		return nil, xerrors.Errorf("SetDiscordChannel: %w", err)
	}
	return &result, nil
}

func (c *Client) SetSpreadsheetID(
	ctx context.Context, puzzle *Puzzle, spreadsheet string,
) (*Puzzle, error) {
	result, err := c.queries.UpdateSpreadsheetID(ctx, UpdateSpreadsheetIDParams{
		ID: puzzle.ID, SpreadsheetID: spreadsheet,
	})
	if err != nil {
		return nil, xerrors.Errorf("UpdateSpreadsheetID: %w", err)
	}
	return &result, nil
}

func (c *Client) SetStatusAndAnswer(
	ctx context.Context, puzzle *Puzzle, status string, answer string,
) (*Puzzle, error) {
	result, err := c.queries.UpdateStatusAndAnswer(ctx, UpdateStatusAndAnswerParams{
		ID: puzzle.ID, Status: status, Answer: answer, Archived: puzzle.IsSolved(),
	})
	if err != nil {
		return nil, xerrors.Errorf("UpdateStatusAndAnswer: %w", err)
	}
	return &result, nil
}

func (c *Client) SetDescription(
	ctx context.Context, puzzle *Puzzle, description string,
) (*Puzzle, error) {
	result, err := c.queries.UpdateDescription(ctx, UpdateDescriptionParams{
		ID: puzzle.ID, Description: description,
	})
	if err != nil {
		return nil, xerrors.Errorf("UpdateDescription: %w", err)
	}
	return &result, nil
}

func (c *Client) SetLocation(
	ctx context.Context, puzzle *Puzzle, location string,
) (*Puzzle, error) {
	result, err := c.queries.UpdateLocation(ctx, UpdateLocationParams{
		ID: puzzle.ID, Location: location,
	})
	if err != nil {
		return nil, xerrors.Errorf("UpdateLocation: %w", err)
	}
	return &result, nil
}

func (c *Client) SetBotFields(ctx context.Context, puzzle *Puzzle) (*Puzzle, error) {

	result, err := c.queries.UpdateArchived(ctx, UpdateArchivedParams{
		ID: puzzle.ID, Archived: puzzle.ShouldArchive(),
	})
	if err != nil {
		return nil, xerrors.Errorf("UpdateArchived: %w", err)
	}
	return &result, nil
}

func (c *Client) SetVoiceRoom(
	ctx context.Context, puzzle *Puzzle, channel *discordgo.Channel,
) (*Puzzle, error) {
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
	return &result, nil
}
