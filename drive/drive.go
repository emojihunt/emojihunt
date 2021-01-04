package drive

import (
	"context"
	"fmt"
	"log"
	"strings"

	"google.golang.org/api/sheets/v4"
)

type Drive struct {
	// ID of the Google Sheet. From the sheets URL: docs.google.com/spreadsheets/d/[ID]/edit
	sheetID string
	// Name of the sheet with puzzle metadata.
	sheetName string

	svc *sheets.Service
}

func New(ctx context.Context, sheetID string) (*Drive, error) {
	sheetsService, err := sheets.NewService(ctx)
	if err != nil {
		return nil, err
	}
	return &Drive{sheetID: sheetID, svc: sheetsService}, nil
}

type Round struct {
	Name  string
	Emoji string
}

// TODO: how should we support extending Status?
type Status string

const (
	Working    Status = "🏅Solved"
	Abandoned  Status = "✍️Working,🗑️Abandoned,🏅Solved,🤦‍♀️Backsolved"
	Solved     Status = "✍️Working,🗑️Abandoned,🏅Solved,🤦‍♀️Backsolved"
	Backsolved Status = "✍️Working,🗑️Abandoned,🏅Solved,🤦‍♀️Backsolved"
)

var allStatuses = []Status{Working, Abandoned, Solved, Backsolved}

type PuzzleInfo struct {
	Round      Round
	Name       string
	Answer     string
	Meta       bool
	PuzzleURL  string
	DocURL     string
	DiscordURL string
	Status     Status
	// TODO: should this be an enum somehow?
	Tags []string
	// Row is 0-indexed, unlike A1 notation sheet rows (1-indexed).
	Row int
}

func parsePuzzleInfo(row []*sheets.CellData, rowNum int) (PuzzleInfo, error) {
	if len(row) != 9 {
		return PuzzleInfo{}, fmt.Errorf("wrong number of fields in row: %v", row)
	}

	return PuzzleInfo{
		Round:      Round{Emoji: row[0].FormattedValue},
		Name:       row[1].FormattedValue,
		Answer:     row[2].FormattedValue,
		Meta:       row[3].FormattedValue != "",
		PuzzleURL:  row[4].Hyperlink,
		DocURL:     row[5].Hyperlink,
		DiscordURL: row[6].Hyperlink,
		Status:     Status(row[7].FormattedValue),
		Tags:       strings.Split(row[8].FormattedValue, ","),
		Row:        rowNum,
	}, nil
}

func (d *Drive) ReadFullSheet() ([]PuzzleInfo, error) {
	req := &sheets.GetSpreadsheetByDataFilterRequest{
		DataFilters:     []*sheets.DataFilter{{A1Range: fmt.Sprintf("'%s'!A2:I", d.sheetName)}},
		IncludeGridData: true,
	}

	s, err := d.svc.Spreadsheets.GetByDataFilter(d.sheetID, req).Do()
	if err != nil {
		return nil, fmt.Errorf("error getting spreadsheet: %v", err)
	}
	if len(s.Sheets) != 1 || len(s.Sheets[0].Data) != 1 {
		return nil, fmt.Errorf("unexpected number of sheets or data ranges returned: %v", s.Sheets)
	}
	var infos []PuzzleInfo
	start := s.Sheets[0].Data[0].StartRow
	for i, row := range s.Sheets[0].Data[0].RowData {
		pi, err := parsePuzzleInfo(row.Values, int(start)+i)
		if len(row.Values) != 9 {
			return nil, fmt.Errorf("unexpected row size %d: %v", len(row.Values), row.Values)
		}
		if err != nil {
			return nil, fmt.Errorf("error parsing puzzle from: %+v", row)
		}
		if pi.Name != "" {
			infos = append(infos, pi)
		}
	}

	return infos, nil
}

func (d *Drive) CreateSheet(ctx context.Context, name string) (url string, err error) {
	log.Printf("would create sheet for %v", name)
	// TODO: implement
	return "https://docs.google.com/spreadsheets/d/1SgvhTBeVdyTMrCR0wZixO3O0lErh4vqX0--nBpSfYT8/edit", nil
}
