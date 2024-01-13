package drive

import (
	"context"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/xerrors"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const (
	DevRootFolderID = "1KNcBa-GjA9Uz8LJ5OYs_zfWRvFRDVu1d"

	// ID of this year's root folder (e.g. the "emoji hunt/2023" folder)
	ProdRootFolderID = "1qrztjHhZuM0FT09HKHuRiwvGHs1Lepcz"
)

type Client struct {
	drive        *drive.Service
	sheets       *sheets.Service
	rootFolderID string
}

func NewClient(ctx context.Context, prod bool) *Client {
	raw, ok := os.LookupEnv("GOOGLE_CREDENTIALS")
	if !ok {
		log.Panicf("GOOGLE_CREDENTIALS is required")
	}
	credentials, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		log.Panicf("GOOGLE_CREDENTIALS is not valid base64")
	}
	sheetsService, err := sheets.NewService(ctx, option.WithCredentialsJSON(credentials))
	if err != nil {
		log.Panicf("sheets.NewService: %s", err)
	}
	driveService, err := drive.NewService(ctx, option.WithCredentialsJSON(credentials))
	if err != nil {
		log.Panicf("drive.NewService: %s", err)
	}

	var rootFolderID = DevRootFolderID
	if prod {
		rootFolderID = ProdRootFolderID
	}
	return &Client{
		drive:        driveService,
		sheets:       sheetsService,
		rootFolderID: rootFolderID,
	}
}

func (c *Client) CreateSheet(ctx context.Context, name string) (id string, err error) {
	sheet, err := withRetry("sheets.Create", func() (*sheets.Spreadsheet, error) {
		return c.sheets.Spreadsheets.Create(&sheets.Spreadsheet{
			Sheets: []*sheets.Sheet{
				{Properties: &sheets.SheetProperties{
					Title: name, ForceSendFields: []string{"SheetId"},
				}},
			},
		}).Context(ctx).Do()
	})
	if err != nil {
		return "", err
	}
	return sheet.SpreadsheetId, nil
}

func (c *Client) SetSheetTitle(ctx context.Context, sheetID, title string) error {
	_, err := withRetry("sheets.BatchUpdate", func() (*sheets.BatchUpdateSpreadsheetResponse, error) {
		var req = &sheets.BatchUpdateSpreadsheetRequest{
			Requests: []*sheets.Request{{
				UpdateSpreadsheetProperties: &sheets.UpdateSpreadsheetPropertiesRequest{
					Fields: "title",
					Properties: &sheets.SpreadsheetProperties{
						Title: title,
					},
				},
			}},
		}
		return c.sheets.Spreadsheets.BatchUpdate(sheetID, req).Context(ctx).Do()
	})
	return err
}

func (c *Client) CreateFolder(ctx context.Context, name string) (id string, err error) {
	file, err := withRetry("drive.Files.Create", func() (*drive.File, error) {
		return c.drive.Files.Create(&drive.File{
			Name:     name,
			MimeType: "application/vnd.google-apps.folder",
			Parents:  []string{c.rootFolderID},
		}).Context(ctx).Do()
	})
	if err != nil {
		return "", err
	}
	return file.Id, nil
}

func (c *Client) SetSheetFolder(ctx context.Context, sheetID, folderID string) error {
	_, err := withRetry("drive.AddParents", func() (*drive.File, error) {
		return c.drive.Files.Update(sheetID, nil).EnforceSingleParent(true).
			AddParents(folderID).Context(ctx).Do()
	})
	return err
}

func (c *Client) SetFolderName(ctx context.Context, folderID, name string) error {
	_, err := withRetry("drive.Files.Update", func() (*drive.File, error) {
		return c.drive.Files.Update(folderID, &drive.File{
			Name:     name,
			MimeType: "application/vnd.google-apps.folder",
		}).EnforceSingleParent(true).AddParents(c.rootFolderID).Context(ctx).Do()
	})
	return err
}

func withRetry[T any](name string, request func() (T, error)) (result T, err error) {
	for w := 5 * time.Second; w <= 30*time.Second; w += 5 * time.Second {
		result, err = request()
		if err == nil {
			return
		}
		var ge *googleapi.Error
		if ok := errors.As(err, &ge); !ok ||
			!(ge.Code == http.StatusTooManyRequests || ge.Code >= 500) {
			break
		}
		log.Printf("%s: retrying HTTP %d: %#v", name, ge.Code, ge.Message)
		time.Sleep(w)
	}
	err = xerrors.Errorf("%s: %w", name, err)
	return
}
