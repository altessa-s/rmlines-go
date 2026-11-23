// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package handwriting

import (
	"context"
	"errors"
	"math"
	"slices"

	rmlines "github.com/altessa-s/rmlines-go"
)

type point struct{ x, y float64 }

func (p point) minus(q point) point   { return point{p.x - q.x, p.y - q.y} }
func (p point) dot(q point) float64   { return p.x*q.x + p.y*q.y }
func (p point) cross(q point) float64 { return p.x*q.y - p.y*q.x }

func between(a, b point, t float64) point {
	if t == 1 {
		return b
	}
	return point{a.x + t*(b.x-a.x), a.y + t*(b.y-a.y)}
}

type trajectory struct {
	X []float64 `json:"x"`
	Y []float64 `json:"y"`
}

// Retain page drawing order: an eraser can only cut trajectories already seen.
func visibleInk(ctx context.Context, strokes []rmlines.Stroke) ([]trajectory, error) {
	var lines [][]point
	for _, s := range strokes {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if len(s.Points) == 0 || s.Tool.IsHighlighter() {
			continue
		}
		points := make([]point, len(s.Points))
		for i, p := range s.Points {
			points[i] = point{float64(p.X), float64(p.Y)}
			if !finite(points[i].x) || !finite(points[i].y) {
				return nil, errors.New("handwriting: invalid stroke coordinates")
			}
		}
		if !s.Tool.IsEraser() {
			lines = append(lines, points)
			continue
		}
		var masks []region
		if s.Tool == rmlines.ToolEraserArea {
			if len(points) >= 3 {
				masks = append(masks, polygon(points))
			}
		} else {
			var previousRadius float64
			for i, p := range points {
				width := float64(s.Points[i].Width) / rmlines.InkUnitsPerRawWidth
				if !finite(width) {
					return nil, errors.New("handwriting: invalid eraser width")
				}
				radius := min(max(width, rmlines.InkMinimumStrokeWidth), rmlines.InkMaximumStrokeWidth) / 2
				masks = append(masks, disk(p, radius))
				if i > 0 && p != points[i-1] {
					a := points[i-1]
					d := p.minus(a)
					length := math.Hypot(d.x, d.y)
					nx, ny := -d.y/length, d.x/length
					masks = append(masks, polygon([]point{
						{a.x + nx*previousRadius, a.y + ny*previousRadius}, {p.x + nx*radius, p.y + ny*radius},
						{p.x - nx*radius, p.y - ny*radius}, {a.x - nx*previousRadius, a.y - ny*previousRadius},
					}))
				}
				previousRadius = radius
			}
		}
		if len(masks) == 0 {
			continue
		}
		var next [][]point
		for _, line := range lines {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			next = append(next, cut(line, masks)...)
		}
		lines = next
	}
	ink := make([]trajectory, len(lines))
	for i, line := range lines {
		t := trajectory{X: make([]float64, len(line)), Y: make([]float64, len(line))}
		for j, p := range line {
			t.X[j] = 10 * p.x
			t.Y[j] = 10 * p.y
		}
		ink[i] = t
	}
	return ink, nil
}
func finite(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }

type bounds struct{ xmin, ymin, xmax, ymax float64 }

func segmentBounds(a, b point) bounds {
	return bounds{min(a.x, b.x), min(a.y, b.y), max(a.x, b.x), max(a.y, b.y)}
}
func (a bounds) overlaps(b bounds) bool {
	return a.xmin <= b.xmax && b.xmin <= a.xmax && a.ymin <= b.ymax && b.ymin <= a.ymax
}

type region struct {
	bounds
	center   point
	radius   float64
	vertices []point
}

func disk(p point, r float64) region {
	return region{bounds: bounds{p.x - r, p.y - r, p.x + r, p.y + r}, center: p, radius: r}
}
func polygon(points []point) region {
	b := segmentBounds(points[0], points[0])
	for _, p := range points[1:] {
		b.xmin = min(b.xmin, p.x)
		b.xmax = max(b.xmax, p.x)
		b.ymin = min(b.ymin, p.y)
		b.ymax = max(b.ymax, p.y)
	}
	return region{bounds: b, vertices: points}
}

func (r region) covers(p point) bool {
	if !r.overlaps(segmentBounds(p, p)) {
		return false
	}
	if r.vertices == nil {
		d := p.minus(r.center)
		return d.dot(d) <= r.radius*r.radius
	}
	inside := false
	a := r.vertices[len(r.vertices)-1]
	for _, b := range r.vertices {
		if p.minus(a).cross(b.minus(a)) == 0 && segmentBounds(a, b).overlaps(segmentBounds(p, p)) {
			return true
		}
		if (a.y > p.y) != (b.y > p.y) && p.x < a.x+(p.y-a.y)*(b.x-a.x)/(b.y-a.y) {
			inside = !inside
		}
		a = b
	}
	return inside
}

func addCut(ts []float64, t float64) []float64 {
	if t > 0 && t < 1 {
		return append(ts, t)
	}
	return ts
}

// Intersections with the pen segment partition it into wholly covered or visible intervals.
func (r region) crossings(ts []float64, a, b point) []float64 {
	d := b.minus(a)
	if r.vertices == nil {
		q := a.minus(r.center)
		dd := d.dot(d)
		proj := q.dot(d)
		disc := dd*r.radius*r.radius - q.cross(d)*q.cross(d)
		if disc >= 0 && dd > 0 {
			root := math.Sqrt(disc)
			ts = addCut(ts, (-proj-root)/dd)
			ts = addCut(ts, (-proj+root)/dd)
		}
		return ts
	}
	p := r.vertices[len(r.vertices)-1]
	for _, q := range r.vertices {
		e := q.minus(p)
		den := d.cross(e)
		delta := p.minus(a)
		if den != 0 {
			u := delta.cross(d) / den
			if u >= 0 && u <= 1 {
				ts = addCut(ts, delta.cross(e)/den)
			}
		} else if delta.cross(d) == 0 && d.dot(d) > 0 {
			ts = addCut(ts, delta.dot(d)/d.dot(d))
			ts = addCut(ts, q.minus(a).dot(d)/d.dot(d))
		}
		p = q
	}
	return ts
}

func cut(line []point, masks []region) [][]point {
	if len(line) == 1 {
		for _, mask := range masks {
			if mask.covers(line[0]) {
				return nil
			}
		}
		return [][]point{line}
	}
	var parts [][]point
	var run []point
	var candidates []region
	var ts []float64
	for i, b := range line[1:] {
		a := line[i]
		box := segmentBounds(a, b)
		candidates = candidates[:0]
		ts = append(ts[:0], 0, 1)
		for _, m := range masks {
			if m.overlaps(box) {
				candidates = append(candidates, m)
				ts = m.crossings(ts, a, b)
			}
		}
		slices.Sort(ts)
		ts = slices.Compact(ts)
		for j, end := range ts[1:] {
			start := ts[j]
			mid := between(a, b, (start+end)/2)
			hidden := false
			for _, m := range candidates {
				if m.covers(mid) {
					hidden = true
					break
				}
			}
			if hidden {
				if len(run) > 0 {
					parts = append(parts, run)
					run = nil
				}
				continue
			}
			if len(run) == 0 {
				run = append(run, between(a, b, start))
			}
			last := between(a, b, end)
			if last != run[len(run)-1] {
				run = append(run, last)
			}
		}
	}
	if len(run) > 0 {
		parts = append(parts, run)
	}
	return parts
}
