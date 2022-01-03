package client

import (
	"fmt"
	"strings"

	"github.com/gauravjsingh/emojihunt/schema"
	"github.com/mehanizm/airtable"
)

type Airtable struct {
	table *airtable.Table
}

const pageSize = 100 // most records returned per list request

const pendingSuffix = " [pending]" // puzzle name suffix for auto-added puzzles

// FYI the Airtable library has a built-in rate limiter that will block if we
// exceed 4 requests per second. This will keep us under Airtable's 5
// requests-per-second limit, which is important because if we break that limit
// we get suspended for 30 seconds.

func NewAirtable(apiKey, dbName, tableName string) *Airtable {
	client := airtable.NewClient(apiKey)
	table := client.GetTable(dbName, tableName)
	return &Airtable{table}
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
	if len(response.Records) != 1 {
		return nil, fmt.Errorf("expected 1 record, got: %#v", response.Records)
	}
	return air.parseRecord(response.Records[0])
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

func (air *Airtable) parseRecord(record *airtable.Record) (*schema.Puzzle, error) {
	round := schema.ParseRound(air.stringField(record, "Round"))
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
		Name:   puzzleName,
		Answer: air.stringField(record, "Answer"),
		Round:  round,
		Status: status,

		AirtableRecord: record,
		PuzzleURL:      air.stringField(record, "Puzzle URL"),
		SpreadsheetID:  air.stringField(record, "Spreadsheet ID"),
		DiscordChannel: air.stringField(record, "Discord Channel"),

		Pending:       pending,
		LastBotStatus: lastBotStatus,
		Archived:      air.boolField(record, "Archived"),
		OriginalURL:   air.stringField(record, "Original URL"),
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
