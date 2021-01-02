package drive

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/api/sheets/v4"
)

type Drive struct {
	// ID of the Google Sheet. From the sheets URL: docs.google.com/spreadsheets/d/[ID]/edit
	sheetID string

	svc *sheets.Service
}

func New(ctx context.Context, apiKey, sheetID string) (*Drive, error) {
	sheetsService, err := sheets.NewService(ctx)
	if err != nil {
		return nil, err
	}
	return &Drive{sheetID: sheetID, svc: sheetsService}, nil
}

func (d *Drive) DumpData() error {
	// Get all data for the sheet.
	req := &sheets.GetSpreadsheetByDataFilterRequest{
		IncludeGridData: true,
	}

	s, err := d.svc.Spreadsheets.GetByDataFilter(d.sheetID, req).Do()
	if err != nil {
		return fmt.Errorf("error getting spreadsheet: %v", err)
	}
	log.Print("logging spreadsheet")
	for _, row := range s.Sheets[0].Data[0].RowData {
		var out []string
		for _, v := range row.Values {
			out = append(out, v.FormattedValue)
		}
		log.Print(out)
	}

	return nil
}
