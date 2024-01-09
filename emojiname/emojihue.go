package emojiname

import (
	_ "embed"
	"encoding/json"

	"golang.org/x/xerrors"
)

//go:embed emoji-hues.json
var hueData []byte
var hues = loadHues()

func loadHues() map[string]uint8 {
	var hues = make(map[string]uint8)

	var data [][]interface{}
	err := json.Unmarshal(hueData, &data)
	if err != nil {
		panic(xerrors.Errorf("loadHues unmarshal: %w", err))
	}
	for _, line := range data {
		for _, emoji := range line[1:] {
			hues[emoji.(string)] = uint8(line[0].(float64))
		}
	}
	return hues
}

func EmojiHue(emoji string) uint8 {
	return hues[emoji]
}
