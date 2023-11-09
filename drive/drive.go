package drive

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"golang.org/x/xerrors"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type Config struct {
	RootFolderID   string      `json:"root_folder_id"`
	ServiceAccount interface{} `json:"service_account"`
}

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

func NewClient(ctx context.Context, config *Config) (*Client, error) {
	rawServiceAccount, err := json.Marshal(config.ServiceAccount)
	if err != nil {
		return nil, xerrors.Errorf("json.Marshal: %w", err)
	}

	sheetsService, err := sheets.NewService(ctx, option.WithCredentialsJSON(rawServiceAccount))
	if err != nil {
		return nil, xerrors.Errorf("sheets.NewService: %w", err)
	}
	driveService, err := drive.NewService(ctx, option.WithCredentialsJSON(rawServiceAccount))
	if err != nil {
		return nil, xerrors.Errorf("drive.NewService: %w", err)
	}

	return &Client{
		rootFolderID:   config.RootFolderID,
		roundFolderIDs: make(map[string]string),
		sheets:         sheetsService,
		drive:          driveService,
	}, nil
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

func (c *Client) SetSheetFolder(ctx context.Context, sheetID, roundName string) error {
	folderID, err := c.roundFolder(ctx, roundName)
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
