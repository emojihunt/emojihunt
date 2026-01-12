package drive

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/xerrors"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/driveactivity/v2"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const (
	// All puzzle spreadsheets are created as copies of this template
	TemplateSheetID = "1WIpd27BwvZCQm355t5u1bDD2TefktZXuZRpRNtO6Tjo"
)

type Client struct {
	drive         *drive.Service
	driveActivity *driveactivity.Service
	sheets        *sheets.Service
	rootFolderID  string
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

	rootFolderID, ok := os.LookupEnv("GOOGLE_DRIVE_FOLDER")
	if !ok {
		log.Panicf("GOOGLE_DRIVE_FOLDER is required")
	}
	_, err = withRetry("drive.CheckFolder", func() (*drive.File, error) {
		return driveService.Files.Get(rootFolderID).Context(ctx).Do()
	})
	if err != nil {
		log.Panicf("no such Google Drive item: %s", rootFolderID)
	}

	driveActivityService, err := driveactivity.NewService(ctx, option.WithCredentialsJSON(credentials))
	if err != nil {
		log.Panicf("driveactivity.NewService: %s", err)
	}

	return &Client{
		drive:         driveService,
		driveActivity: driveActivityService,
		sheets:        sheetsService,
		rootFolderID:  rootFolderID,
	}
}

func (c *Client) CreateSheet(ctx context.Context, name string, folder string) (id string, err error) {
	file, err := withRetry("sheets.Create", func() (*drive.File, error) {
		return c.drive.Files.Copy(TemplateSheetID, &drive.File{
			Parents: []string{folder},
		}).Context(ctx).Do()
	})
	if err != nil {
		return "", err
	}
	err = c.SetSheetTitle(ctx, file.Id, name)
	if err != nil {
		return "", err
	}
	return file.Id, nil
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
	if err != nil {
		return err
	}
	withRetry("sheets.BatchUpdate", func() (*sheets.BatchUpdateSpreadsheetResponse, error) {
		var req = &sheets.BatchUpdateSpreadsheetRequest{
			Requests: []*sheets.Request{{
				UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
					Fields: "title",
					Properties: &sheets.SheetProperties{
						Title: title,
					},
				},
			}},
		}
		return c.sheets.Spreadsheets.BatchUpdate(sheetID, req).Context(ctx).Do()
	})
	return nil // this one is best-effort, in case Tab 0 has been deleted
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

func (c *Client) QueryActivity(ctx context.Context) (map[string]time.Time, error) {
	var pageToken string
	var result = make(map[string]time.Time)
	var limit = time.Now().Add(-100 * time.Minute)
	for range 16 {
		raw, err := withRetry("drive.Activity.Query", func() (*driveactivity.QueryDriveActivityResponse, error) {
			return c.driveActivity.Activity.Query(
				&driveactivity.QueryDriveActivityRequest{
					AncestorName: "items/" + c.rootFolderID,
					ConsolidationStrategy: &driveactivity.ConsolidationStrategy{
						Legacy: &driveactivity.Legacy{},
					},
					Filter:    fmt.Sprintf("time > %d AND detail.action_detail_case:(EDIT COMMENT)", limit.UnixMilli()),
					PageSize:  512,
					PageToken: pageToken,
				},
			).Context(ctx).Do()
		})
		if err != nil {
			return nil, err
		}

		for _, activity := range raw.Activities {
			var ts string
			if activity.Timestamp != "" {
				ts = activity.Timestamp
			} else if activity.TimeRange != nil {
				ts = activity.TimeRange.EndTime
			} else {
				log.Printf("drive: no timestamp found on activity")
				continue
			}
			timestamp, err := time.Parse(time.RFC3339, ts)
			if err != nil {
				return nil, err
			}

			var editedByHuman bool
			for _, user := range activity.Actors {
				if user.User == nil {
					continue // ???
				} else if user.User.KnownUser != nil && user.User.KnownUser.IsCurrentUser {
					continue // skip our own (huntbot's) edits
				} else {
					editedByHuman = true
					break
				}
			}
			if !editedByHuman {
				continue
			}

			for _, target := range activity.Targets {
				if target.DriveItem == nil {
					continue
				}
				id, found := strings.CutPrefix(target.DriveItem.Name, "items/")
				if !found {
					log.Printf("drive: unrecognized drive name %q", target.DriveItem.Name)
					continue
				}
				if previous, ok := result[id]; !ok || timestamp.After(previous) {
					result[id] = timestamp
				}
			}
		}
		if raw.NextPageToken == "" {
			break
		}
		pageToken = raw.NextPageToken
	}
	return result, nil
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
