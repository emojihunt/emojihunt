package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/emojihunt/emojihunt/state/db"
	"golang.org/x/xerrors"
)

const (
	discoveryConfigSetting  = "discovery_config"
	enabledSetting          = "discovery_enabled"
	discoveredRoundsSetting = "discovered_rounds"
	reminderSetting         = "reminder_timestamp"
	syncEpochSetting        = "sync_epoch"
)

func (c *Client) IsEnabled(ctx context.Context) bool {
	data, err := c.readSetting(ctx, enabledSetting)
	if err != nil {
		panic(err)
	}
	var enabled bool = true // default
	if len(data) > 0 {
		err = json.Unmarshal(data, &enabled)
		if err != nil {
			panic(xerrors.Errorf("IsEnabled unmarshal: %w", err))
		}
	}
	return enabled
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
	c.DiscoveryChange <- true
	return true
}

func (c *Client) DiscoveredRounds(ctx context.Context) (map[string]DiscoveredRound, error) {
	data, err := c.readSetting(ctx, discoveredRoundsSetting)
	if err != nil {
		return nil, err
	}
	var rounds map[string]DiscoveredRound
	if len(data) > 0 {
		err = json.Unmarshal(data, &rounds)
		if err != nil {
			return nil, xerrors.Errorf("DiscoveredRounds unmarshal: %w", err)
		}
	}
	if rounds == nil {
		rounds = make(map[string]DiscoveredRound)
	}
	return rounds, nil
}

func (c *Client) SetDiscoveredRounds(ctx context.Context, rounds map[string]DiscoveredRound) error {
	// Concurrency rule: this setting is only written from the discovery poller's
	// round creation worker goroutine.
	return c.writeSetting(ctx, discoveredRoundsSetting, rounds)
}

func (c *Client) ReminderTimestamp(ctx context.Context) (time.Time, error) {
	data, err := c.readSetting(ctx, reminderSetting)
	if err != nil {
		return time.Time{}, err
	}
	var reminder time.Time
	if len(data) > 0 {
		err = json.Unmarshal(data, &reminder)
		if err != nil {
			return time.Time{}, xerrors.Errorf("ReminderTimestamp unmarshal: %w", err)
		}
	}
	return reminder, nil
}

func (c *Client) SetReminderTimestamp(ctx context.Context, reminder time.Time) error {
	// Concurrency rule: this setting is only written from the reminder bot's
	// worker goroutine.
	return c.writeSetting(ctx, reminderSetting, reminder)
}

func (c *Client) IncrementSyncEpoch(ctx context.Context) (int64, error) {
	// Concurrency rule: this setting is only written on startup.
	data, err := c.readSetting(ctx, syncEpochSetting)
	if err != nil {
		return 0, err
	}
	var epoch int64 = 1 //default
	if len(data) > 0 {
		err = json.Unmarshal(data, &epoch)
		if err != nil {
			return 0, xerrors.Errorf("IncrementSyncEpoch unmarshal: %w", err)
		}
	}
	return epoch, c.writeSetting(ctx, syncEpochSetting, epoch+1)
}

func (c *Client) readSetting(ctx context.Context, key string) ([]byte, error) {
	data, err := c.queries.GetSetting(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, xerrors.Errorf("GetSetting: %w", err)
	}
	return data, err
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
