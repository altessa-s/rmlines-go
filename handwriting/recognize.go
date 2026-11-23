// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

// Package handwriting converts selected reMarkable pen trajectories using the account's cloud gateway.
package handwriting

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	rmlines "github.com/altessa-s/rmlines-go"
)

// Box is a word's bounding rectangle in JIIX millimetres.
type Box struct{ X, Y, Width, Height float64 }
type Word struct {
	Label  string `json:"label"`
	Bounds *Box   `json:"bounding-box"`
}
type Result struct {
	Text  string
	Words []Word
}

// Recognize sends the supplied ink to gateway with the caller's authenticated client.
// Pass strokes in drawing order, in a common scene coordinate system.
func Recognize(ctx context.Context, client *http.Client, gateway string, strokes []rmlines.Stroke, language string) (Result, error) {
	ink, err := visibleInk(ctx, strokes)
	if err != nil {
		return Result{}, err
	}
	if len(ink) == 0 {
		return Result{}, nil
	}
	if client == nil {
		return Result{}, errors.New("handwriting: authenticated HTTP client required")
	}
	endpoint, err := url.Parse(gateway)
	if err != nil || endpoint.Scheme != "https" || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return Result{}, errors.New("handwriting: invalid HTTPS gateway")
	}
	if language == "" {
		language = "en_US"
	}
	config := map[string]any{
		"lang": strings.ReplaceAll(language, "-", "_"),
		"export": map[string]any{"jiix": map[string]any{
			"strokes": false, "text": map[string]bool{"chars": false, "words": true},
		}},
		"raw-content": map[string]any{"recognition": map[string]bool{"text": true, "shape": false}},
	}
	body, err := json.Marshal(map[string]any{
		"xDPI": 2280, "yDPI": 2280, "width": 14040, "height": 18720,
		"contentType": "Raw Content", "configuration": config,
		"strokeGroups": []any{map[string]any{"strokes": ink}},
	})
	if err != nil {
		return Result{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 70*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.JoinPath("convert/v1/handwriting").String(), bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Accept", "application/vnd.myscript.jiix")
	req.Header.Set("Content-Type", "application/json")
	sender := *client
	sender.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := sender.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("handwriting: HTTP %d", resp.StatusCode)
	}
	const maxSize = 32 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSize+1))
	if err != nil {
		return Result{}, err
	}
	if len(data) > maxSize {
		return Result{}, errors.New("handwriting: response too large")
	}
	return decode(data)
}

type element struct {
	Type     string
	Label    *string
	Words    []Word
	Elements []element
}

func decode(data []byte) (Result, error) {
	var root element
	if err := json.Unmarshal(data, &root); err != nil {
		return Result{}, err
	}
	if root.Type != "Text" && root.Type != "Raw Content" {
		return Result{}, errors.New("handwriting: invalid JIIX root")
	}
	var words []Word
	var labels []string
	var visit func(element) error
	visit = func(e element) error {
		if e.Type == "Text" {
			if e.Label == nil {
				return errors.New("handwriting: text element has no label")
			}
			labels = append(labels, *e.Label)
			words = append(words, e.Words...)
			return nil
		}
		for _, child := range e.Elements {
			if err := visit(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(root); err != nil {
		return Result{}, err
	}
	return Result{Text: strings.Join(labels, "\n"), Words: words}, nil
}
