package drive

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

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
	roundFolderIDs map[string]string

	// hold while accessing roundFolderIDs
	mu sync.Mutex

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
		sheetID:        sheetID,
		rootFolderID:   rootFolderID,
		roundFolderIDs: make(map[string]string),
		sheets:         sheetsService,
		drive:          driveService,
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
		// TODO: figure out how to decide the round name
		Round:      Round{Emoji: row[0].FormattedValue, Name: row[0].FormattedValue},
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

func (d *Drive) CreateSheet(ctx context.Context, name, roundName string) (url string, err error) {
	log.Printf("Creating sheet for %v", name)

	sheet := &sheets.Spreadsheet{
		Properties: &sheets.SpreadsheetProperties{
			Title: name,
		},
	}

	sheet, err = d.sheets.Spreadsheets.Create(sheet).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("unable to create sheet for %q: %v", name, err)
	}

	folderID, err := d.roundFolder(ctx, roundName)
	if err != nil {
		return "", err
	}

	log.Println(folderID)

	_, err = d.drive.Files.Update(sheet.SpreadsheetId, nil).AddParents(folderID).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("unable to add sheet for %q to folder for round %q: %v", name, roundName, err)
	}

	return sheet.SpreadsheetUrl, nil
}

const folderMimeType = "application/vnd.google-apps.folder"

func (d *Drive) roundFolder(ctx context.Context, name string) (id string, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if id = d.roundFolderIDs[name]; id != "" {
		return id, nil
	}

	query := "mimeType='" + folderMimeType + "' and " +
		"'" + d.rootFolderID + "' in parents and " +
		"name = '" + name + "'"
	list, err := d.drive.Files.List().Q(query).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("couldn't query for existing folder for round %q: %v", name, err)
	}

	var file *drive.File
	switch len(list.Files) {
	case 0:
		file = &drive.File{
			Name:     name,
			MimeType: folderMimeType,
			Parents:  []string{d.rootFolderID},
		}
		file, err = d.drive.Files.Create(file).Context(ctx).Do()
		if err != nil {
			return "", fmt.Errorf("couldn't create folder for round %q: %v", name, err)
		}
	case 1:
		file = list.Files[0]
	default:
		return "", fmt.Errorf("found multiple folders for round %q", name)
	}

	d.roundFolderIDs[name] = file.Id
	return file.Id, nil
}
