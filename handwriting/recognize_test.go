package handwriting

import (
	"context"
	"encoding/json"
	rmlines "github.com/altessa-s/rmlines-go"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecognize(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/convert/v1/handwriting" || r.Header.Get("Accept") != "application/vnd.myscript.jiix" {
			t.Error("invalid conversion request")
		}
		var body struct {
			Configuration struct{ Lang string }
			StrokeGroups  []struct{ Strokes []struct{ X, Y []float64 } }
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Configuration.Lang != "en_US" || len(body.StrokeGroups) != 1 || len(body.StrokeGroups[0].Strokes) != 1 {
			t.Error("invalid stroke group")
			return
		}
		points := body.StrokeGroups[0].Strokes[0]
		if len(points.X) != 1 || points.X[0] != 20 || points.Y[0] != 30 {
			t.Error("invalid scene scale")
		}
		_, _ = w.Write([]byte(`{"type":"Raw Content","elements":[{"type":"Text","label":"hello","words":[{"label":"hello","bounding-box":{"x":11.2543859,"y":13.8166666,"width":15.7444086,"height":8.8513155}}]}]}`))
	}))
	defer srv.Close()
	got, err := Recognize(context.Background(), srv.Client(), srv.URL, []rmlines.Stroke{{Points: []rmlines.StrokePoint{{X: 2, Y: 3}}}}, "en-US")
	if err != nil || got.Text != "hello" || len(got.Words) != 1 || got.Words[0].Bounds == nil || got.Words[0].Bounds.Width != 15.7444086 {
		t.Fatalf("result %v, error %v", got, err)
	}
}
func TestNonText(t *testing.T) {
	got, err := decode([]byte(`{"type":"Raw Content","elements":[{"type":"Raw Content","kind":"non-text"}]}`))
	if err != nil || got.Text != "" {
		t.Fatal(got, err)
	}
	if _, err = decode([]byte("{}")); err == nil {
		t.Fatal("accepted non-JIIX")
	}
}

func TestDecodeRootType(t *testing.T) {
	for _, input := range []string{`{"type":"Wrong","elements":[]}`, `{"elements":[]}`} {
		if _, err := decode([]byte(input)); err == nil {
			t.Errorf("accepted %s", input)
		}
	}
	if _, err := decode([]byte(`{"type":"Raw Content"}`)); err != nil {
		t.Fatal(err)
	}
}
