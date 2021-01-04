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
	Abandoned  Status = "🗑️Abandoned"
	Solved     Status = "🏅Solved"
	Backsolved Status = "🤦‍♀️Backsolved"
)

var allStatuses = []Status{Working, Abandoned, Solved, Backsolved}

const (
	puzzleIcon  string = "🌎"
	docIcon     string = "✏️"
	discordIcon string = "💬"
)

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

func (p *PuzzleInfo) sheetRow() int {
	return p.Row + 1
}

func (p *PuzzleInfo) metaFormula() string {
	if !p.Meta {
		return ""
	}
	return fmt.Sprintf(`=CONCATENATE(COUNTIFS($A$2:$A, $A%d, $C$2:$C, "<>0", $B$2:$B, "<>"&$B%d), "/", COUNTIF($A$2:$A, $A%d)-1)`,
		p.sheetRow(), p.sheetRow(), p.sheetRow())
}

func formatURL(url, icon string) string {
	return fmt.Sprintf(`=HYPERLINK("%s","%s")`, url, icon)
}

func (p *PuzzleInfo) URLsAsValueRange(sheetName string) *sheets.ValueRange {
	vr := &sheets.ValueRange{
		Range: fmt.Sprintf("'%s'!E%d:G%d", sheetName, p.sheetRow(), p.sheetRow()),
		Values: [][]interface{}{{
			formatURL(p.PuzzleURL, puzzleIcon),
			formatURL(p.DocURL, docIcon),
			formatURL(p.DiscordURL, discordIcon),
		}},
	}
	return vr
}

func (p *PuzzleInfo) AsValueRange(sheetName string) *sheets.ValueRange {
	vr := &sheets.ValueRange{
		Range: fmt.Sprintf("'%s'!A%d:I%d", sheetName, p.sheetRow(), p.sheetRow()),
		Values: [][]interface{}{{
			p.Round.Emoji,
			p.Name,
			strings.ToUpper(p.Answer),
			p.metaFormula(),
			formatURL(p.PuzzleURL, puzzleIcon),
			formatURL(p.DocURL, docIcon),
			formatURL(p.DiscordURL, discordIcon),
			p.Status,
			strings.Join(p.Tags, ","),
		}},
	}
	return vr
}

func parsePuzzleInfo(row []*sheets.CellData, rowNum int) (*PuzzleInfo, error) {
	if len(row) != 9 {
		return nil, fmt.Errorf("wrong number of fields in row: %v", row)
	}

	return &PuzzleInfo{
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

func (d *Drive) ReadFullSheet() ([]*PuzzleInfo, error) {
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
	var infos []*PuzzleInfo
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

func (d *Drive) UpdateCell(ctx context.Context, a1CellLocation string, value interface{}) error {
	_, err := d.sheets.Spreadsheets.Values.Update(d.sheetID, fmt.Sprintf("'%s'!%s", d.sheetName, a1CellLocation),
		&sheets.ValueRange{Values: [][]interface{}{{value}}}).ValueInputOption("USER_ENTERED").Context(ctx).Do()
	return err
}

func (d *Drive) SetDocURL(ctx context.Context, p *PuzzleInfo) error {
	a1Loc := fmt.Sprintf("F%d", p.Row+1)
	hyperlink := fmt.Sprintf(`=HYPERLINK("%s","✏️")`, p.DocURL)
	return d.UpdateCell(ctx, a1Loc, hyperlink)
}

func (d *Drive) SetDiscordURL(ctx context.Context, p *PuzzleInfo) error {
	a1Loc := fmt.Sprintf("G%d", p.Row+1)
	hyperlink := fmt.Sprintf(`=HYPERLINK("%s","💬")`, p.DiscordURL)
	return d.UpdateCell(ctx, a1Loc, hyperlink)
}

func (d *Drive) UpdatePuzzle(ctx context.Context, p *PuzzleInfo) error {
	a1Range := fmt.Sprintf("A%d:I%d", p.Row+1, p.Row+1)

	_, err := d.sheets.Spreadsheets.Values.Update(d.sheetID, fmt.Sprintf("'%s'!%s", d.sheetName, a1Range),
		p.AsValueRange(d.sheetName)).ValueInputOption("USER_ENTERED").Context(ctx).Do()
	return err
}

func (d *Drive) UpdateAllURLs(ctx context.Context, ps []*PuzzleInfo) error {
	req := &sheets.BatchUpdateValuesRequest{ValueInputOption: "USER_ENTERED"}
	for _, p := range ps {
		req.Data = append(req.Data, p.URLsAsValueRange(d.sheetName))
	}
	_, err := d.sheets.Spreadsheets.Values.BatchUpdate(d.sheetID, req).Context(ctx).Do()
	return err
}

func (d *Drive) MarkSheetSolved(ctx context.Context, sheetURL string) error {
	if !strings.HasPrefix(sheetURL, "https://docs.google.com/spreadsheets/d/") {
		return fmt.Errorf("not a valid sheet URL: %q", sheetURL)
	}
	parts := strings.Split(sheetURL, "/")
	id := parts[len(parts)-2]
	sheet, err := d.sheets.Spreadsheets.Get(id).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("couldn't find sheet at %q: %v", sheetURL, err)
	}

	if strings.HasPrefix(sheet.Properties.Title, "[SOLVED]") {
		return nil // nothing to do
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{{
			UpdateSpreadsheetProperties: &sheets.UpdateSpreadsheetPropertiesRequest{
				Fields: "title",
				Properties: &sheets.SpreadsheetProperties{
					Title: "[SOLVED] " + sheet.Properties.Title,
				},
			},
		}},
	}

	_, err = d.sheets.Spreadsheets.BatchUpdate(id, req).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("couldn't update sheet %q title: %v", sheet.Properties.Title, err)
	}

	return nil
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
