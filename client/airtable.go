package client

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gauravjsingh/emojihunt/schema"
	"github.com/mehanizm/airtable"
)

type AirtableConfig struct {
	APIKey  string `json:"api_key"`
	BaseID  string `json:"base_id"`
	TableID string `json:"table_id"`
}

type Airtable struct {
	baseID, tableID string
	table           *airtable.Table
}

const pageSize = 100 // most records returned per list request

const pendingSuffix = " [pending]" // puzzle name suffix for auto-added puzzles

// FYI the Airtable library has a built-in rate limiter that will block if we
// exceed 4 requests per second. This will keep us under Airtable's 5
// requests-per-second limit, which is important because if we break that limit
// we get suspended for 30 seconds.

func NewAirtable(config *AirtableConfig) *Airtable {
	client := airtable.NewClient(config.APIKey)
	table := client.GetTable(config.BaseID, config.TableID)
	return &Airtable{config.BaseID, config.TableID, table}
}

func (air *Airtable) ListRecords() ([]schema.Puzzle, error) {
	var infos []schema.Puzzle
	var offset = ""
	for {
		response, err := air.table.GetRecords().
			PageSize(pageSize).
			WithOffset(offset).
			Do()
		if err != nil {
			return nil, err
		}

		for _, record := range response.Records {
			if record.Deleted {
				// Skip deleted records? I think this field is only used in
				// response to DELETE requests, but let's check it just in case.
				continue
			}
			info, err := air.parseRecord(record)
			if err != nil {
				return nil, err
			}
			infos = append(infos, *info)
		}

		if response.Offset != "" {
			// More records exist, continue to next request
			offset = response.Offset
		} else {
			// All done, return all records
			return infos, nil
		}
	}
}

func (air *Airtable) FindByID(id string) (*schema.Puzzle, error) {
	record, err := air.table.GetRecord(id)
	if err != nil {
		return nil, err
	}
	return air.parseRecord(record)
}

func (air *Airtable) FindByDiscordChannel(channel string) (*schema.Puzzle, error) {
	response, err := air.table.GetRecords().
		WithFilterFormula(fmt.Sprintf("{Discord Channel}='%s'", channel)).
		Do()
	if err != nil {
		return nil, err
	}
	if len(response.Records) < 1 {
		return nil, nil
	} else if len(response.Records) > 1 {
		return nil, fmt.Errorf("expected 0 or 1 record, got: %#v", response.Records)
	}
	return air.parseRecord(response.Records[0])
}

func (air *Airtable) FindWithVoiceRoomEvent() ([]*schema.Puzzle, error) {
	response, err := air.table.GetRecords().
		WithFilterFormula("{Voice Room Event}!=''").
		Do()
	if err != nil {
		return nil, err
	} else if response.Offset != "" {
		// This shouldn't happen, but if it does we fail instead of spending too
		// much of our rate limit on paginated requests.
		return nil, fmt.Errorf("airtable query failed: too many records have a voice room")
	}

	var puzzles []*schema.Puzzle
	for _, record := range response.Records {
		puzzle, err := air.parseRecord(record)
		if err != nil {
			return nil, err
		}
		puzzles = append(puzzles, puzzle)
	}
	return puzzles, nil
}

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
		"Archived":        status.IsSolved(),
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

func (air *Airtable) UpdateVoiceRoomEvent(puzzle *schema.Puzzle, voiceRoomEvent string) (*schema.Puzzle, error) {
	record, err := puzzle.AirtableRecord.UpdateRecordPartial(map[string]interface{}{
		"Voice Room Event": voiceRoomEvent,
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

func (air *Airtable) EditURL(puzzle *schema.Puzzle) string {
	return fmt.Sprintf(
		"https://airtable.com/%s/%s/%s",
		air.baseID, air.tableID, puzzle.AirtableRecord.ID,
	)
}

func (air *Airtable) parseRecord(record *airtable.Record) (*schema.Puzzle, error) {
	var rounds schema.Rounds
	if raw, ok := record.Fields["Round"]; ok {
		switch v := raw.(type) {
		case string:
			if raw != "" {
				rounds = append(rounds, schema.ParseRound(raw.(string)))
			}
		case []interface{}:
			for _, r := range raw.([]interface{}) {
				rounds = append(rounds, schema.ParseRound(r.(string)))
			}
		default:
			return nil, fmt.Errorf("airtable: can't handle round field of type %T", v)
		}
	}
	sort.Sort(rounds)

	status, err := schema.ParsePrettyStatus(air.stringField(record, "Status"))
	if err != nil {
		return nil, err
	}

	lastBotStatus, err := schema.ParseTextStatus(air.stringField(record, "Last Bot Status"))
	if err != nil {
		return nil, err
	}

	puzzleName := air.stringField(record, "Name")
	pending := false
	if strings.HasSuffix(puzzleName, pendingSuffix) {
		// Ideally the "pending" status would be stored in a separate field, but
		// we want it to be obvious to humans viewing the Airtable.
		puzzleName = strings.TrimSuffix(puzzleName, pendingSuffix)
		pending = true
	}

	return &schema.Puzzle{
		Name:        puzzleName,
		Answer:      air.stringField(record, "Answer"),
		Rounds:      rounds,
		Status:      status,
		Description: air.stringField(record, "Description"),
		Notes:       air.stringField(record, "Notes"),

		AirtableRecord: record,
		PuzzleURL:      air.stringField(record, "Puzzle URL"),
		SpreadsheetID:  air.stringField(record, "Spreadsheet ID"),
		DiscordChannel: air.stringField(record, "Discord Channel"),

		Pending:        pending,
		LastBotStatus:  lastBotStatus,
		Archived:       air.boolField(record, "Archived"),
		OriginalURL:    air.stringField(record, "Original URL"),
		VoiceRoomEvent: air.stringField(record, "Voice Room Event"),
	}, nil
}

func (air *Airtable) stringField(record *airtable.Record, field string) string {
	if value, ok := record.Fields[field]; !ok {
		// Airtable omits empty records from their JSON responses, so we can't
		// actually tell if we've typo'd the field name.
		return ""
	} else {
		return value.(string)
	}
}

func (air *Airtable) boolField(record *airtable.Record, field string) bool {
	if value, ok := record.Fields[field]; !ok {
		return false
	} else {
		return value.(bool)
	}
}
