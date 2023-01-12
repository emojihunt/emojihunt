package client

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/emojihunt/emojihunt/schema"
	"github.com/mehanizm/airtable"
)

type AirtableConfig struct {
	APIKey    string `json:"api_key"`
	BaseID    string `json:"base_id"`
	TableID   string `json:"table_id"`
	BotUserID string `json:"bot_user_id"`
}

type Airtable struct {
	BotUserID         string
	ModifyGracePeriod time.Duration

	baseID  string
	tableID string
	table   *airtable.Table

	// A map of Airtable Record ID -> puzzle mutex. The puzzle mutex should be
	// held while reading or writing the puzzle, and should be acquired before
	// the voice room mutex (if needed).
	mutexes *sync.Map

	// Mutex mutex protects channelToRecord. It should be held briefly when
	// updating the map. We should never perform an operation that could block,
	// like acquiring another lock or making an API call, while holding mutex.
	mutex           *sync.Mutex
	channelToRecord map[string]string
}

const (
	defaultGracePeriod = 3 * time.Second // delay before records are picked up by ListPuzzlesToAction
	pageSize           = 100             // most records returned per list request
)

// FYI the Airtable library has a built-in rate limiter that will block if we
// exceed 4 requests per second. This will keep us under Airtable's 5
// requests-per-second limit, which is important because if we break that limit
// we get suspended for 30 seconds.

func NewAirtable(config *AirtableConfig) *Airtable {
	return &Airtable{
		BotUserID:         config.BotUserID,
		ModifyGracePeriod: defaultGracePeriod,

		baseID:  config.BaseID,
		tableID: config.TableID,
		table: airtable.
			NewClient(config.APIKey).
			GetTable(config.BaseID, config.TableID),
		mutexes:         &sync.Map{},
		mutex:           &sync.Mutex{},
		channelToRecord: make(map[string]string),
	}
}

func (air *Airtable) EditURL(puzzle *schema.Puzzle) string {
	return fmt.Sprintf(
		"https://airtable.com/%s/%s/%s",
		air.baseID, air.tableID, puzzle.AirtableRecord.ID,
	)
}

func (air *Airtable) parseRecord(record *airtable.Record, unlock func()) (*schema.Puzzle, error) {
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

	var lastModifiedBy string
	if value, ok := record.Fields["Last Modified By"]; ok {
		lastModifiedBy = value.(map[string]interface{})["id"].(string)
	}

	return &schema.Puzzle{
		Name:         air.stringField(record, "Name"),
		Answer:       air.stringField(record, "Answer"),
		Rounds:       rounds,
		Status:       status,
		Description:  air.stringField(record, "Description"),
		Location:     air.stringField(record, "Location"),
		NameOverride: air.stringField(record, "Puzzle Name Override"),

		AirtableRecord: record,
		PuzzleURL:      air.stringField(record, "Puzzle URL"),
		SpreadsheetID:  air.stringField(record, "Spreadsheet ID"),
		DiscordChannel: air.stringField(record, "Discord Channel"),

		LastBotStatus: lastBotStatus,
		Archived:      air.boolField(record, "Archived"),
		LastBotSync:   air.timeField(record, "Last Bot Sync"),

		OriginalURL: air.stringField(record, "Original URL"),
		VoiceRoom:   air.stringField(record, "Voice Room"),
		Reminder:    air.timeField(record, "Reminder"),

		LastModified:   air.timeField(record, "Last Modified"),
		LastModifiedBy: lastModifiedBy,

		Unlock: unlock,
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

func (air *Airtable) timeField(record *airtable.Record, field string) *time.Time {
	if value, ok := record.Fields[field]; !ok || value == "" {
		return nil
	} else {
		if t, err := time.Parse(time.RFC3339, value.(string)); err != nil {
			panic(fmt.Errorf("error parsing time: %v", err))
		} else {
			return &t
		}
	}
}
