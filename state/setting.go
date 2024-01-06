package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/emojihunt/emojihunt/state/db"
	"golang.org/x/xerrors"
)

const (
	enabledSetting          = "discovery_enabled"
	discoveredRoundsSetting = "discovered_rounds"
	reminderSetting         = "reminder_timestamp"
	syncEpochSetting        = "sync_epoch"
)

func (c *Client) IsEnabled(ctx context.Context) bool {
	if raw, err := c.readSetting(ctx, enabledSetting); err != nil {
		panic(err)
	} else if v, ok := raw.(bool); !ok {
		return true // default
	} else {
		return v
	}
}

func (c *Client) EnableDiscovery(ctx context.Context, enabled bool) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.IsEnabled(ctx) == enabled {
		return false
	}
	if err := c.writeSetting(ctx, enabledSetting, enabled); err != nil {
		panic(err)
	}
	c.DiscoveryChange <- enabled
	return true
}

func (c *Client) DiscoveredRounds(ctx context.Context) (map[string]DiscoveredRound, error) {
	if raw, err := c.readSetting(ctx, discoveredRoundsSetting); err != nil {
		return nil, err
	} else if v, ok := raw.(map[string]DiscoveredRound); !ok {
		return make(map[string]DiscoveredRound), nil // default
	} else {
		return v, nil
	}
}

func (c *Client) SetDiscoveredRounds(ctx context.Context, rounds map[string]DiscoveredRound) error {
	// Concurrency rule: this setting is only written from the discovery poller's
	// round creation worker goroutine.
	return c.writeSetting(ctx, discoveredRoundsSetting, rounds)
}

func (c *Client) ReminderTimestamp(ctx context.Context) (time.Time, error) {
	raw, err := c.readSetting(ctx, reminderSetting)
	if err != nil {
		return time.Time{}, err
	}
	var previous int64 = 0 // default
	if s, ok := raw.(string); ok {
		previous, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			return time.Time{}, err
		}
	}
	return time.Unix(previous, 0), nil
}

func (c *Client) SetReminderTimestamp(ctx context.Context, timestamp time.Time) error {
	// Concurrency rule: this setting is only written from the reminder bot's
	// worker goroutine.
	return c.writeSetting(ctx, reminderSetting, strconv.FormatInt(timestamp.Unix(), 10))
}

func (c *Client) IncrementSyncEpoch(ctx context.Context) (int64, error) {
	// Concurrency rule: this setting is only written on startup.
	raw, err := c.readSetting(ctx, syncEpochSetting)
	if err != nil {
		return 0, err
	}
	var previous int64 = 1 // default
	if s, ok := raw.(string); ok {
		previous, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, err
		}
	}
	return previous, c.writeSetting(ctx, syncEpochSetting,
		strconv.FormatInt(previous+1, 10))
}

func (c *Client) readSetting(ctx context.Context, key string) (interface{}, error) {
	data, err := c.queries.GetSetting(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, xerrors.Errorf("GetSetting: %w", err)
	}
	var result interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, xerrors.Errorf("setting unmarshal: %w", err)
	}
	return result, nil
}

func (c *Client) writeSetting(ctx context.Context, key string, value interface{}) error {
	data, err := json.MarshalIndent(&value, "", "")
	if err != nil {
		return xerrors.Errorf("setting marshal: %w", err)
	}
	err = c.queries.UpdateSetting(ctx, db.UpdateSettingParams{
		Key: key, Value: data,
	})
	if err != nil {
		return xerrors.Errorf("UpdateSetting: %w", err)
	}
	return nil
}
