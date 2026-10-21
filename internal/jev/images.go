package jev

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"regexp"
)

const VisionModel = "jev-1.14.0"

type ImageAsset struct {
	Type   string `json:"type"`
	Format string `json:"format"`
	Data   string `json:"data_base64"`
}

var imageReference = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)

func validateImages(call Call) error {
	if len(call.Images) == 0 {
		return nil
	}
	if call.Priority != PriorityBackground || len(call.Images) > 3 {
		return fmt.Errorf("%w: background images only", ErrRequest)
	}
	total := 0
	for ref, asset := range call.Images {
		total += len(asset.Data)
		if total > 4<<20 || !imageReference.MatchString(ref) || asset.Type != "image" || (asset.Format != "jpg" && asset.Format != "png") {
			return fmt.Errorf("%w: image asset", ErrRequest)
		}
		data, err := base64.StdEncoding.DecodeString(asset.Data)
		if err != nil {
			return fmt.Errorf("%w: image encoding", ErrRequest)
		}
		cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || (format != "jpeg" && format != "png") || (asset.Format == "jpg" && format != "jpeg") || (asset.Format == "png" && format != "png") || cfg.Width < 280 || cfg.Height < 280 || cfg.Width > 1568 || cfg.Height > 1568 {
			return fmt.Errorf("%w: image dimensions or format", ErrRequest)
		}
	}
	return nil
}

func (c *Client) callModel(call Call) string {
	if len(call.Images) > 0 {
		return VisionModel
	}
	return c.model
}
