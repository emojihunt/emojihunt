package client

import (
	"encoding/json"

	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/state"
	"github.com/gorilla/websocket"
	"golang.org/x/xerrors"
)

func ReadMessage(ws *websocket.Conn) (state.LiveMessage, error) {
	var raw struct {
		Event state.EventType `json:"event"`
		Data  json.RawMessage `json:"data"`
	}
	err := ws.ReadJSON(&raw)
	if err != nil {
		return nil, err
	}

	var dst any
	switch raw.Event {
	case state.EventTypeDiscord:
		dst = new(discord.AblyMessage)
	case state.EventTypeSettings:
		dst = new(SettingsMessage)
	case state.EventTypeSync:
		dst = new(state.AblySyncMessage)
	default:
		return nil, xerrors.Errorf("unhandled event type: %#v", raw.Event)
	}
	err = json.Unmarshal(raw.Data, &dst)
	return dst.(state.LiveMessage), err
}

func WriteMessage(ws *websocket.Conn, msg state.LiveMessage) error {
	return ws.WriteJSON(
		struct {
			Event state.EventType `json:"event"`
			Data  any             `json:"data"`
		}{
			Event: msg.EventType(),
			Data:  msg,
		},
	)
}
