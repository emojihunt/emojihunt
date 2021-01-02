package drive

import (
	"context"
	"flag"
	"fmt"
	"log"

	"google.golang.org/api/sheets/v4"
)

var id = flag.String("sheet_id", "1SgvhTBeVdyTMrCR0wZixO3O0lErh4vqX0--nBpSfYT8", "the id of the sheet to use")

func ConnectToDrive(apiKey string) error {
	ctx := context.Background()
	sheetsService, err := sheets.NewService(ctx)
	if err != nil {
		return err
	}

	// Get all data for the sheet.
	req := &sheets.GetSpreadsheetByDataFilterRequest{
		IncludeGridData: true,
	}

	s, err := sheetsService.Spreadsheets.GetByDataFilter(*id, req).Do()
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
