package drive

import (
	"context"
	"encoding/base64"
	"log"
	"os"

	"golang.org/x/xerrors"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const (
	DevRootFolderID = "1KNcBa-GjA9Uz8LJ5OYs_zfWRvFRDVu1d"

	// ID of this year's root folder (e.g. the "emoji hunt/2023" folder)
	ProdRootFolderID = "1gg7CZmoteIjLrk2ifHcc8FSG5HVo2vpt"
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
	sheet, err := c.sheets.Spreadsheets.Create(&sheets.Spreadsheet{}).Context(ctx).Do()
	if err != nil {
		return "", xerrors.Errorf("sheets.Create (%q): %w", name, err)
	}
	return sheet.SpreadsheetId, nil
}

func (c *Client) SetSheetTitle(ctx context.Context, sheetID, title string) error {
	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{{
			UpdateSpreadsheetProperties: &sheets.UpdateSpreadsheetPropertiesRequest{
				Fields: "title",
				Properties: &sheets.SpreadsheetProperties{
					Title: title,
				},
			},
		}},
	}
	_, err := c.sheets.Spreadsheets.
		BatchUpdate(sheetID, req).
		Context(ctx).
		Do()
	if err != nil {
		return xerrors.Errorf("sheets.BatchUpdate (%s): %w", sheetID, err)
	}
	return nil
}

func (c *Client) CreateFolder(ctx context.Context, name string) (id string, err error) {
	file, err := c.drive.Files.Create(&drive.File{
		Name:     name,
		MimeType: "application/vnd.google-apps.folder",
		Parents:  []string{c.rootFolderID},
	}).Context(ctx).Do()
	if err != nil {
		return "", xerrors.Errorf("drive.Files.Create (%s): %w", name, err)
	}
	return file.Id, nil
}

func (c *Client) SetSheetFolder(ctx context.Context, sheetID, folderID string) error {
	_, err := c.drive.Files.Update(sheetID, nil).
		EnforceSingleParent(true).
		AddParents(folderID).
		Context(ctx).
		Do()
	if err != nil {
		return xerrors.Errorf("drive.AddParents (%s): %w", sheetID, err)
	}
	return nil
}
