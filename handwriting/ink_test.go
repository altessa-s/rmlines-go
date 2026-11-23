package handwriting

import (
	"context"
	"reflect"
	"testing"

	rm "github.com/altessa-s/rmlines-go"
)

func TestErasersClipOnlyEarlierInk(t *testing.T) {
	line := rm.Stroke{Tool: rm.ToolBallpoint2, Points: []rm.StrokePoint{{X: 0}, {X: 10}}}
	marker := rm.Stroke{Tool: rm.ToolHighlighter2, Color: rm.ColorHighlighterGray, Points: line.Points}
	for _, test := range []struct {
		name   string
		eraser rm.Stroke
	}{
		{"masked", rm.Stroke{Tool: rm.ToolMaskedEraser, Points: []rm.StrokePoint{{X: 5, Y: -3, Width: 8}, {X: 5, Y: 3, Width: 8}}}},
		{"ordinary", rm.Stroke{Tool: rm.ToolEraser, Points: []rm.StrokePoint{{X: 5, Width: 8}}}},
		{"area", rm.Stroke{Tool: rm.ToolEraserArea, Points: []rm.StrokePoint{{X: 4, Y: -2}, {X: 6, Y: -2}, {X: 6, Y: 2}, {X: 4, Y: 2}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			ink, err := visibleInk(context.Background(), []rm.Stroke{line, marker, test.eraser, line})
			want := []trajectory{{X: []float64{0, 40}, Y: []float64{0, 0}}, {X: []float64{60, 100}, Y: []float64{0, 0}}, {X: []float64{0, 100}, Y: []float64{0, 0}}}
			if err != nil || !reflect.DeepEqual(ink, want) {
				t.Fatalf("got %+v, %v; want %+v", ink, err, want)
			}
		})
	}
}

func TestFullyErasedPageStaysLocal(t *testing.T) {
	p := rm.StrokePoint{X: 5, Y: 7, Width: 8}
	result, err := Recognize(context.Background(), nil, "", []rm.Stroke{
		{Tool: rm.ToolFineliner2, Points: []rm.StrokePoint{p}},
		{Tool: rm.ToolMaskedEraser, Points: []rm.StrokePoint{p}},
	}, "en-US")
	if err != nil || result.Text != "" {
		t.Fatalf("%+v %v", result, err)
	}
}
