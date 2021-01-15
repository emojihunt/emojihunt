package drive

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"
)

type Drive struct {
	// ID of the central tracking sheet. From the sheets URL:
	// docs.google.com/spreadsheets/d/[ID]/edit
	sheetID string

	// Name of the sheet (tab) within the tracking spreadsheet
	// with puzzle metadata.
	puzzlesTab string

	// Name of the sheet (tab) within the tracking spreadsheet
	// with round metadata.
	roundsTab string

	// ID of this year's root folder (e.g. "emoji hunt/2021")
	rootFolderID string

	// hold while accessing roundFolderIDs, puzzles maps
	mu sync.Mutex
	// cache of round name to folder ID
	roundFolderIDs map[string]string
	// map from channel URL to puzzle name
	chanToPuzzles map[string]string

	sheets *sheets.Service
	drive  *drive.Service
}

func New(ctx context.Context, sheetID, puzzlesTab, roundsTab, rootFolderID string) (*Drive, error) {
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
		puzzlesTab:     puzzlesTab,
		roundsTab:      roundsTab,
		rootFolderID:   rootFolderID,
		roundFolderIDs: make(map[string]string),
		sheets:         sheetsService,
		drive:          driveService,
	}, nil
}

type Round struct {
	Name  string
	Emoji string
	Color *sheets.Color
}

func (r *Round) TwemojiURL() string {
	codePoints := make([]string, 0)
	for _, runeValue := range r.Emoji {
		codePoints = append(codePoints, fmt.Sprintf("%04x", runeValue))
	}
	return fmt.Sprintf("https://twemoji.maxcdn.com/2/72x72/%s.png", strings.Join(codePoints, "-"))
}

func (r *Round) IntColor() int {
	red := int(r.Color.Red * 255)
	green := int(r.Color.Green * 255)
	blue := int(r.Color.Blue * 255)
	return (red << 16) + (green << 8) + blue
}

// TODO: how should we support extending Status?
type Status string

const (
	Working    Status = "🏅Solved"
	Abandoned  Status = "🗑️Abandoned"
	Solved     Status = "🏅Solved"
	Backsolved Status = "🤦‍♀️Backsolved"
)

func (s Status) IsSolved() bool {
	return s == Solved || s == Backsolved
}

func (s Status) Pretty() string {
	if string(s) == "" {
		return "Not Started"
	} else {
		return string(s)
	}
}

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

// TODO: this might be more accurately named round formula; it counts # of unsolved puzzles in the round.
func (p *PuzzleInfo) metaFormula() string {
	if !p.Meta {
		return ""
	}
	return fmt.Sprintf(`=CONCATENATE(COUNTIFS($A$2:$A, $A%d, $C$2:$C, "<>0", $B$2:$B, "<>"&$B%d), "/", COUNTIF($A$2:$A, $A%d)-1)`,
		p.sheetRow(), p.sheetRow(), p.sheetRow())
}

func formatURL(url, icon string) string {
	if url == "" {
		return ""
	}
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

func parseRoundInfo(row []*sheets.CellData) (*Round, error) {
	if len(row) != 2 {
		return nil, fmt.Errorf("wrong number of fields in round row: %v", row)
	}

	return &Round{
		Emoji: row[0].FormattedValue,
		Name:  row[1].FormattedValue,
		Color: row[0].EffectiveFormat.BackgroundColor,
	}, nil
}

func parsePuzzleInfo(row []*sheets.CellData, rounds map[string]*Round, rowNum int) (*PuzzleInfo, error) {
	if len(row) != 9 {
		return nil, fmt.Errorf("wrong number of fields in puzzle row: %v", row)
	}

	round, ok := rounds[row[0].FormattedValue]
	if !ok {
		round = &Round{Emoji: row[0].FormattedValue, Name: row[0].FormattedValue}
	}

	return &PuzzleInfo{
		Round:  *round,
		Name:   row[1].FormattedValue,
		Answer: row[2].FormattedValue,
		// Metas have a formula for the number of unsolved puzzles in column D.
		Meta:       row[3].FormattedValue != "",
		PuzzleURL:  row[4].Hyperlink,
		DocURL:     row[5].Hyperlink,
		DiscordURL: row[6].Hyperlink,
		Status:     Status(row[7].FormattedValue),
		Tags:       strings.Split(row[8].FormattedValue, ","),
		Row:        rowNum,
	}, nil
}

func (d *Drive) ReadFullSheet(ctx context.Context) ([]*PuzzleInfo, error) {
	req := &sheets.GetSpreadsheetByDataFilterRequest{
		DataFilters: []*sheets.DataFilter{
			{A1Range: fmt.Sprintf("'%s'!A2:I", d.puzzlesTab)},
			{A1Range: fmt.Sprintf("'%s'!A2:B", d.roundsTab)},
		},
		IncludeGridData: true,
	}

	s, err := d.sheets.Spreadsheets.GetByDataFilter(d.sheetID, req).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("error getting spreadsheet: %v", err)
	}
	if len(s.Sheets) != 2 || len(s.Sheets[0].Data) != 1 || len(s.Sheets[1].Data) != 1 {
		return nil, fmt.Errorf("unexpected number of sheets or data ranges returned: %v", s.Sheets)
	}
	var infos []*PuzzleInfo

	rounds := make(map[string]*Round)
	for _, row := range s.Sheets[1].Data[0].RowData {
		if round, err := parseRoundInfo(row.Values); err != nil {
			return nil, fmt.Errorf("error parsing round from row: %+v: %v", row, err)
		} else if round.Emoji != "" && round.Name != "" {
			rounds[round.Emoji] = round
		}
	}

	if err := d.SetConditionalFormatting(ctx, rounds); err != nil {
		return nil, fmt.Errorf("error setting conditional formatting: %v", err)
	}

	start := s.Sheets[0].Data[0].StartRow
	for i, row := range s.Sheets[0].Data[0].RowData {
		pi, err := parsePuzzleInfo(row.Values, rounds, int(start)+i)
		if err != nil {
			return nil, fmt.Errorf("error parsing puzzle from row: %+v: %v", row, err)
		}
		if pi.Name != "" {
			infos = append(infos, pi)
		}
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	chanToPuzzles := make(map[string]string)
	for _, i := range infos {
		if i.DiscordURL != "" {
			chanToPuzzles[i.DiscordURL] = i.Name
		}
	}
	d.chanToPuzzles = chanToPuzzles
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
	_, err := d.sheets.Spreadsheets.Values.Update(d.sheetID, fmt.Sprintf("'%s'!%s", d.puzzlesTab, a1CellLocation),
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
	a1Range := fmt.Sprintf("A%d:I%d", p.sheetRow(), p.sheetRow())

	_, err := d.sheets.Spreadsheets.Values.Update(d.sheetID, fmt.Sprintf("'%s'!%s", d.puzzlesTab, a1Range),
		p.AsValueRange(d.puzzlesTab)).ValueInputOption("USER_ENTERED").Context(ctx).Do()
	return err
}

func (d *Drive) UpdateAllURLs(ctx context.Context, ps []*PuzzleInfo) error {
	req := &sheets.BatchUpdateValuesRequest{ValueInputOption: "USER_ENTERED"}
	for _, p := range ps {
		req.Data = append(req.Data, p.URLsAsValueRange(d.puzzlesTab))
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

	query := fmt.Sprintf("mimeType='%s' and '%s' in parents and name = '%s'", folderMimeType, d.rootFolderID, name)
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

func (d *Drive) PuzzleForChannelURL(chanURL string) (string, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	p, ok := d.chanToPuzzles[chanURL]
	return p, ok
}

var formulaEmojiRE = regexp.MustCompile(`\$A2=\"(.*)\"`)

func findConditionalFormattingIndices(s *sheets.Spreadsheet) (map[string][]int, error) {
	indices := make(map[string][]int)
	for i, c := range s.Sheets[0].ConditionalFormats {
		if c.BooleanRule.Condition.Type == "CUSTOM_FORMULA" {
			matches := formulaEmojiRE.FindStringSubmatch(c.BooleanRule.Condition.Values[0].UserEnteredValue)
			if len(matches) == 0 {
				continue
			}
			emoji := matches[1]
			indices[emoji] = append(indices[emoji], i)
		}
	}

	return indices, nil
}

func addConditionalFormattingRequests(r *Round) []*sheets.Request {
	ranges := []*sheets.GridRange{{
		// Skip the categories for round formatting.
		StartRowIndex: 1,
	}}
	unsolvedRule := &sheets.BooleanRule{
		Condition: &sheets.BooleanCondition{
			Type: "CUSTOM_FORMULA",
			Values: []*sheets.ConditionValue{{
				UserEnteredValue: fmt.Sprintf(`=AND($A2="%s", ISBLANK($C2))`, r.Emoji),
			}},
		},
		Format: &sheets.CellFormat{
			BackgroundColor: r.Color,
		},
	}
	solvedRule := &sheets.BooleanRule{
		Condition: &sheets.BooleanCondition{
			Type: "CUSTOM_FORMULA",
			Values: []*sheets.ConditionValue{{
				UserEnteredValue: fmt.Sprintf(`=AND($A2="%s", NOT(ISBLANK($C2)))`, r.Emoji),
			}},
		},
		Format: &sheets.CellFormat{
			BackgroundColor: r.Color,
			TextFormat: &sheets.TextFormat{
				ForegroundColor: &sheets.Color{Red: 0.6, Green: 0.6, Blue: 0.6},
			},
		},
	}
	reqs := []*sheets.Request{
		{
			AddConditionalFormatRule: &sheets.AddConditionalFormatRuleRequest{
				Rule: &sheets.ConditionalFormatRule{
					BooleanRule: unsolvedRule,
					Ranges:      ranges,
				},
			},
		},
		{
			AddConditionalFormatRule: &sheets.AddConditionalFormatRuleRequest{
				Rule: &sheets.ConditionalFormatRule{
					BooleanRule: solvedRule,
					Ranges:      ranges,
				},
			},
		},
	}

	return reqs
}

func colorEqual(c1, c2 *sheets.Color) bool {
	return c1.Alpha == c2.Alpha && c1.Blue == c2.Blue && c1.Green == c2.Green && c1.Red == c2.Red
}

func (d *Drive) SetConditionalFormatting(ctx context.Context, rs map[string]*Round) error {
	s, err := d.sheets.Spreadsheets.Get(d.sheetID).Context(ctx).Do()
	if err != nil {
		return err
	}

	existing, err := findConditionalFormattingIndices(s)
	if err != nil {
		return err
	}
	req := &sheets.BatchUpdateSpreadsheetRequest{}
	var newFormats []*sheets.Request
	for _, r := range rs {
		indices, ok := existing[r.Emoji]
		if !ok {
			newFormats = append(newFormats, addConditionalFormattingRequests(r)...)
			continue
		}

		for _, i := range indices {
			rule := s.Sheets[0].ConditionalFormats[i]
			if colorEqual(rule.BooleanRule.Format.BackgroundColorStyle.RgbColor, r.Color) {
				continue
			}
			rule.BooleanRule.Format.BackgroundColorStyle.RgbColor = r.Color
			update := &sheets.UpdateConditionalFormatRuleRequest{
				Index: int64(i),
				//NewIndex: int64(i),
				Rule: rule,
				//SheetId:  int64(0),
			}
			req.Requests = append(req.Requests, &sheets.Request{
				UpdateConditionalFormatRule: update,
			})
		}
	}
	req.Requests = append(req.Requests, newFormats...)
	if len(req.Requests) == 0 {
		return nil
	}
	_, err = d.sheets.Spreadsheets.BatchUpdate(d.sheetID, req).Context(ctx).Do()
	return err
}
