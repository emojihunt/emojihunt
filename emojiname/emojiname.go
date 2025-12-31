package emojiname

import (
	_ "embed"
	"encoding/json"
	"log"
	"math/rand"
	"strings"
	"time"

	"golang.org/x/xerrors"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

type Emoji struct {
	// e.g. "THUMBS UP SIGN"
	Name string `json:"name"`
	// e.g. "+1"
	ShortName  string   `json:"short_name"`
	ShortNames []string `json:"short_names"`
	// e.g. "1F44D-1F3FB"
	Unified string `json:"unified"`
}

func contains(haystack []string, needle string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}
	return false
}

func (e *Emoji) weight() float64 {
	switch {
	case strings.HasPrefix(e.ShortName, "flag"):
		return 0.1 // there are ~200
	case strings.HasPrefix(e.ShortName, "clock"):
		return 0.25 // there are 24
	case strings.HasPrefix(e.Name, "SQUARED CJK UNIFIED IDEOGRAPH"):
		return 0
	case strings.HasPrefix(e.Name, "SQUARED KATAKANA"):
		return 0
	case strings.HasPrefix(e.ShortName, "skin-tone"):
		return 0
	case contains(strings.Fields(e.Name), "LATIN"):
		return 0.25
	case contains(strings.Fields(e.Name), "KEYCAP"):
		return 0.25
	default:
		return 1
	}
}

func RandomEmoji(n int) ([]*Emoji, error) {
	ret := make([]*Emoji, n)
	allEmoji, err := Load()
	if err != nil {
		return nil, err
	}

	var totalWeight float64
	for _, e := range allEmoji {
		totalWeight += e.weight()
	}

	for i := 0; i < n; i++ {
		r := rng.Float64() * totalWeight
		for _, e := range allEmoji {
			w := e.weight()
			if r < w {
				ret[i] = e
				break
			}
			r -= w
		}
		if ret[i] == nil {
			log.Printf("fell off end of emoji list")
			ret = append(ret, allEmoji[0])
		}
	}

	return ret, nil
}

// emoji.json from https://github.com/iamcal/emoji-data/blob/master/emoji.json
// Copyright (c) 2013 Cal Henderson and MIT licensed.
//
// Current version: Unicode 16.0
//
//go:embed emoji.json
var rawEmoji []byte
var allEmoji []*Emoji

func Load() ([]*Emoji, error) {
	if allEmoji != nil {
		return allEmoji, nil
	}

	err := json.Unmarshal(rawEmoji, &allEmoji)
	if err != nil {
		return nil, xerrors.Errorf("error reading emoji data: %w", err)
	}

	return allEmoji, nil
}
