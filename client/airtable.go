package client

import (
	"fmt"

	"github.com/gauravjsingh/emojihunt/schema"
	"github.com/mehanizm/airtable"
)

type Airtable struct {
	table *airtable.Table
}

const pageSize = 100 // most records returned per list request

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

func (air *Airtable) UpdateLastBotStatus(puzzle *schema.Puzzle, lastBotStatus schema.Status) (*schema.Puzzle, error) {
	record, err := puzzle.AirtableRecord.UpdateRecordPartial(map[string]interface{}{
		"Last Bot Status": lastBotStatus.Serialize(),
	})
	if err != nil {
		return nil, err
	}
	return air.parseRecord(record)
}

func (air *Airtable) parseRecord(record *airtable.Record) (*schema.Puzzle, error) {
	round := schema.ParseRound(air.stringField(record, "Round"))
	status, err := schema.ParseStatus(air.stringField(record, "Status"))
	if err != nil {
		return nil, err
	}

	lastBotStatus, err := schema.ParseStatus(air.stringField(record, "Last Bot Status"))
	if err != nil {
		return nil, err
	}

	return &schema.Puzzle{
		Name:   air.stringField(record, "Name"),
		Answer: air.stringField(record, "Answer"),
		Round:  round,
		Status: status,

		AirtableRecord: record,
		PuzzleURL:      air.stringField(record, "Puzzle URL"),
		SpreadsheetID:  air.stringField(record, "Spreadsheet ID"),
		DiscordChannel: air.stringField(record, "Discord Channel"),
		LastBotStatus:  lastBotStatus,
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
