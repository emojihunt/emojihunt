package db

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/db/field"
	"golang.org/x/xerrors"
)

func (c *Client) UpdatePuzzle(ctx context.Context, puzzle RawPuzzle) error {
	if err := puzzle.Validate(); err != nil {
		return err
	}
	err := c.queries.UpdatePuzzle(ctx, UpdatePuzzleParams(puzzle))
	if err != nil {
		return xerrors.Errorf("UpdatePuzzle: %w", err)
	}
	return nil
}

func (c *Client) SetDiscordChannel(
	ctx context.Context, puzzle *Puzzle, channel string,
) (*Puzzle, error) {
	err := c.queries.UpdateDiscordChannel(ctx, UpdateDiscordChannelParams{
		ID: puzzle.ID, DiscordChannel: channel,
	})
	if err != nil {
		return nil, xerrors.Errorf("UpdateDiscordChannel: %w", err)
	}
	record, err := c.queries.GetPuzzle(ctx, puzzle.ID)
	if err != nil {
		return nil, xerrors.Errorf("GetPuzzle: %w", err)
	}
	converted := Puzzle(record)
	return &converted, nil
}

func (c *Client) SetSpreadsheetID(
	ctx context.Context, puzzle *Puzzle, spreadsheet string,
) (*Puzzle, error) {
	err := c.queries.UpdateSpreadsheetID(ctx, UpdateSpreadsheetIDParams{
		ID: puzzle.ID, SpreadsheetID: spreadsheet,
	})
	if err != nil {
		return nil, xerrors.Errorf("UpdateSpreadsheetID: %w", err)
	}
	record, err := c.queries.GetPuzzle(ctx, puzzle.ID)
	if err != nil {
		return nil, xerrors.Errorf("GetPuzzle: %w", err)
	}
	converted := Puzzle(record)
	return &converted, nil
}

func (c *Client) SetStatusAndAnswer(
	ctx context.Context, puzzle *Puzzle, status field.Status, answer string,
) (*Puzzle, error) {
	err := c.queries.UpdateStatusAndAnswer(ctx, UpdateStatusAndAnswerParams{
		ID: puzzle.ID, Status: status, Answer: answer, Archived: puzzle.Status.IsSolved(),
	})
	if err != nil {
		return nil, xerrors.Errorf("UpdateStatusAndAnswer: %w", err)
	}
	record, err := c.queries.GetPuzzle(ctx, puzzle.ID)
	if err != nil {
		return nil, xerrors.Errorf("GetPuzzle: %w", err)
	}
	converted := Puzzle(record)
	return &converted, nil
}

func (c *Client) SetNote(
	ctx context.Context, puzzle *Puzzle, note string,
) (*Puzzle, error) {
	err := c.queries.UpdateNote(ctx, UpdateNoteParams{
		ID: puzzle.ID, Note: note,
	})
	if err != nil {
		return nil, xerrors.Errorf("UpdateNote: %w", err)
	}
	record, err := c.queries.GetPuzzle(ctx, puzzle.ID)
	if err != nil {
		return nil, xerrors.Errorf("GetPuzzle: %w", err)
	}
	converted := Puzzle(record)
	return &converted, nil
}

func (c *Client) SetLocation(
	ctx context.Context, puzzle *Puzzle, location string,
) (*Puzzle, error) {
	err := c.queries.UpdateLocation(ctx, UpdateLocationParams{
		ID: puzzle.ID, Location: location,
	})
	if err != nil {
		return nil, xerrors.Errorf("UpdateLocation: %w", err)
	}
	record, err := c.queries.GetPuzzle(ctx, puzzle.ID)
	if err != nil {
		return nil, xerrors.Errorf("GetPuzzle: %w", err)
	}
	converted := Puzzle(record)
	return &converted, nil
}

func (c *Client) SetBotFields(ctx context.Context, puzzle *Puzzle) (*Puzzle, error) {
	err := c.queries.UpdateArchived(ctx, UpdateArchivedParams{
		ID: puzzle.ID, Archived: puzzle.ShouldArchive(),
	})
	if err != nil {
		return nil, xerrors.Errorf("UpdateArchived: %w", err)
	}
	record, err := c.queries.GetPuzzle(ctx, puzzle.ID)
	if err != nil {
		return nil, xerrors.Errorf("GetPuzzle: %w", err)
	}
	converted := Puzzle(record)
	return &converted, nil
}

func (c *Client) SetVoiceRoom(
	ctx context.Context, puzzle *Puzzle, channel *discordgo.Channel,
) (*Puzzle, error) {
	var channelID, channelName string
	if channel != nil {
		channelID = channel.ID
		channelName = channel.Name
	}
	err := c.queries.UpdateVoiceRoom(ctx, UpdateVoiceRoomParams{
		ID: puzzle.ID, VoiceRoom: channelID, Location: channelName,
	})
	if err != nil {
		return nil, xerrors.Errorf("UpdateVoiceRoom: %w", err)
	}
	record, err := c.queries.GetPuzzle(ctx, puzzle.ID)
	if err != nil {
		return nil, xerrors.Errorf("GetPuzzle: %w", err)
	}
	converted := Puzzle(record)
	return &converted, nil
}
