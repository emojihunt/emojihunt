package client

import (
	"context"
	"fmt"
	"log"
	"sync"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type Drive struct {
	// ID of this year's root folder (e.g. "emoji hunt/2021")
	rootFolderID string

	// hold while accessing roundFolderIDs
	mu sync.Mutex
	// cache of round name to folder ID
	roundFolderIDs map[string]string

	sheets *sheets.Service
	drive  *drive.Service
}

func NewDrive(ctx context.Context, rootFolderID string, credentials []byte) (*Drive, error) {
	sheetsService, err := sheets.NewService(ctx, option.WithCredentialsJSON(credentials))
	if err != nil {
		return nil, err
	}
	driveService, err := drive.NewService(ctx, option.WithCredentialsJSON(credentials))
	if err != nil {
		return nil, err
	}

	return &Drive{
		rootFolderID:   rootFolderID,
		roundFolderIDs: make(map[string]string),
		sheets:         sheetsService,
		drive:          driveService,
	}, nil
}

func (d *Drive) CreateSheet(ctx context.Context, name, roundName string) (id string, err error) {
	log.Printf("Creating sheet for %v", name)
	sheet, err := d.sheets.Spreadsheets.Create(&sheets.Spreadsheet{}).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("unable to create sheet for %q: %v", name, err)
	}
	return sheet.SpreadsheetId, nil
}

func (d *Drive) SetSheetTitle(ctx context.Context, sheetID, title string) error {
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
	_, err := d.sheets.Spreadsheets.
		BatchUpdate(sheetID, req).
		Context(ctx).
		Do()
	return err
}

func (d *Drive) SetSheetFolder(ctx context.Context, sheetID, roundName string) error {
	folderID, err := d.roundFolder(ctx, roundName)
	if err != nil {
		return err
	}
	_, err = d.drive.Files.Update(sheetID, nil).
		EnforceSingleParent(true).
		AddParents(folderID).
		Context(ctx).
		Do()
	return err
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
