package airtable

import (
	"github.com/gauravjsingh/emojihunt/schema"
	"github.com/mehanizm/airtable"
)

type Client struct {
	table *airtable.Table
}

const pageSize = 100 // most records returned per list request

// FYI the Airtable library has a built-in rate limiter that will block if we
// exceed 4 requests per second. This will keep us under Airtable's 5
// requests-per-second limit, which is important because if we break that limit
// we get suspended for 30 seconds.

func New(apiKey, dbName, tableName string) *Client {
	client := airtable.NewClient(apiKey)
	table := client.GetTable(dbName, tableName)
	return &Client{table}
}

func (air *Client) ListRecords() ([]schema.Puzzle, error) {
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
			infos = append(infos, *parseRecord(record))
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

func (air *Client) UpdateDiscordChannel(puz *schema.Puzzle, channel string) (*schema.Puzzle, error) {
	record, err := puz.AirtableRecord.UpdateRecordPartial(map[string]interface{}{
		"Discord Channel": channel,
	})
	if err != nil {
		return nil, err
	}
	return parseRecord(record), nil
}

func (air *Client) UpdateSpreadsheetID(puz *schema.Puzzle, spreadsheet string) (*schema.Puzzle, error) {
	record, err := puz.AirtableRecord.UpdateRecordPartial(map[string]interface{}{
		"Spreadsheet ID": spreadsheet,
	})
	if err != nil {
		return nil, err
	}
	return parseRecord(record), nil
}

func parseRecord(record *airtable.Record) *schema.Puzzle {
	return &schema.Puzzle{
		Name:   record.Fields["Name"].(string),
		Answer: record.Fields["Answer"].(string),
		Round:  schema.ParseRound(record.Fields["Round"].(string)),
		Status: schema.ParseStatus(record.Fields["Status"].(string)),

		AirtableRecord: record,
		PuzzleURL:      record.Fields["Puzzle URL"].(string),
		SpreadsheetID:  record.Fields["Spreadsheet ID"].(string),
		DiscordChannel: record.Fields["Discord Channel"].(string),
	}
}
