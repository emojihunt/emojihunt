package drive

import (
	"context"
	"fmt"
	"log"
	"strings"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"
)

type Drive struct {
	// ID of the central tracking sheet. From the sheets URL:
	// docs.google.com/spreadsheets/d/[ID]/edit
	sheetID string
	// Name of the sheet (tab) within that sheet with puzzle metadata.
	sheetName string
	// ID of this year's root folder (e.g. "emoji hunt/2021")
	rootFolderID string

	// cache of round name to folder ID
	folderIDs map[string]string

	sheets *sheets.Service
	drive  *drive.Service
}

func New(ctx context.Context, sheetID, rootFolderID string) (*Drive, error) {
	sheetsService, err := sheets.NewService(ctx)
	if err != nil {
		return nil, err
	}
	driveService, err := drive.NewService(ctx)
	if err != nil {
		return nil, err
	}

	return &Drive{
		sheetID:      sheetID,
		rootFolderID: rootFolderID,
		folderIDs:    make(map[string]string),
		sheets:       sheetsService,
		drive:        driveService,
	}, nil
}

type Round struct {
	Name  string
	Emoji string
}

// TODO: how should we support extending Status?
type Status string

const (
	Working    Status = "🏅Solved"
	Abandoned  Status = "🗑️Abandoned"
	Solved     Status = "🏅Solved"
	Backsolved Status = "🤦‍♀️Backsolved"
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

	s, err := d.sheets.Spreadsheets.GetByDataFilter(d.sheetID, req).Do()
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
	log.Printf("Creating sheet for %v", name)

	// TODO: set sharing/folder
	sheet := &sheets.Spreadsheet{
		Properties: &sheets.SpreadsheetProperties{
			Title: name,
		},
	}

	sheet, err = d.sheets.Spreadsheets.Create(sheet).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("unable to create sheet for %q: %v", name, err)
	}
	return sheet.SpreadsheetUrl, nil
}

func (d *Drive) UpdateCell(ctx context.Context, a1CellLocation string, value interface{}) error {
	_, err := d.sheets.Spreadsheets.Values.Update(d.sheetID, fmt.Sprintf("'%s'!%s", d.sheetName, a1CellLocation),
		&sheets.ValueRange{Values: [][]interface{}{{value}}}).ValueInputOption("USER_ENTERED").Context(ctx).Do()
	return err
}

func (d *Drive) SetDocURL(ctx context.Context, p PuzzleInfo) error {
	a1Loc := fmt.Sprintf("F%d", p.Row+1)
	hyperlink := fmt.Sprintf(`=HYPERLINK("%s","✏️")`, p.DocURL)
	return d.UpdateCell(ctx, a1Loc, hyperlink)
}

func (d *Drive) SetDiscordURL(ctx context.Context, p PuzzleInfo) error {
	a1Loc := fmt.Sprintf("G%d", p.Row+1)
	hyperlink := fmt.Sprintf(`=HYPERLINK("%s","💬")`, p.DiscordURL)
	return d.UpdateCell(ctx, a1Loc, hyperlink)
}
