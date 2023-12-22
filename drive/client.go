package drive

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"sync"

	"golang.org/x/xerrors"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var (
	DevRootFolderID  = "1KNcBa-GjA9Uz8LJ5OYs_zfWRvFRDVu1d"
	ProdRootFolderID = "1gg7CZmoteIjLrk2ifHcc8FSG5HVo2vpt"
)

type Client struct {
	// ID of this year's root folder (e.g. "emoji hunt/2021")
	rootFolderID string

	// hold while accessing roundFolderIDs
	mu sync.Mutex
	// cache of round name to folder ID
	roundFolderIDs map[string]string

	sheets *sheets.Service
	drive  *drive.Service
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
		rootFolderID:   rootFolderID,
		roundFolderIDs: make(map[string]string),
		sheets:         sheetsService,
		drive:          driveService,
	}
}

func (c *Client) CreateSheet(ctx context.Context, name, roundName string) (id string, err error) {
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

func (c *Client) SetSheetFolder(ctx context.Context, sheetID, folderName string) error {
	folderID, err := c.roundFolder(ctx, folderName)
	if err != nil {
		return err
	}
	_, err = c.drive.Files.Update(sheetID, nil).
		EnforceSingleParent(true).
		AddParents(folderID).
		Context(ctx).
		Do()
	if err != nil {
		return xerrors.Errorf("drive.AddParents (%s): %w", sheetID, err)
	}
	return nil
}

const folderMimeType = "application/vnd.google-apps.folder"

func (c *Client) roundFolder(ctx context.Context, name string) (id string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if id = c.roundFolderIDs[name]; id != "" {
		return id, nil
	}

	query := fmt.Sprintf("mimeType='%s' and '%s' in parents and name = '%s'", folderMimeType, c.rootFolderID, name)
	list, err := c.drive.Files.List().Q(query).Context(ctx).Do()
	if err != nil {
		return "", xerrors.Errorf("drive.Query (%q): %w", name, err)
	}

	var file *drive.File
	switch len(list.Files) {
	case 0:
		file = &drive.File{
			Name:     name,
			MimeType: folderMimeType,
			Parents:  []string{c.rootFolderID},
		}
		file, err = c.drive.Files.Create(file).Context(ctx).Do()
		if err != nil {
			return "", xerrors.Errorf("drive.Create (%q): %w", name, err)
		}
	case 1:
		file = list.Files[0]
	default:
		return "", xerrors.Errorf("found multiple folders for round %q", name)
	}

	c.roundFolderIDs[name] = file.Id
	return file.Id, nil
}
