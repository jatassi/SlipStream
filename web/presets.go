package web

import (
	_ "embed"
	"encoding/json"
)

// Preset is a Playwright viewport that the browser tier and verify share.
type Preset struct {
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	DeviceScaleFactor float64 `json:"deviceScaleFactor"`
	IsMobile          bool    `json:"isMobile"`
	HasTouch          bool    `json:"hasTouch"`
}

//go:embed harness/presets.json
var presets []byte

// Presets returns harness/presets.json by preset name.
func Presets() map[string]Preset {
	var p map[string]Preset
	if err := json.Unmarshal(presets, &p); err != nil {
		panic("web: harness/presets.json: " + err.Error())
	}
	return p
}
