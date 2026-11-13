// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package rmlines_test

import (
	"math/rand/v2"
	"runtime"
	"runtime/metrics"
	"strings"
	"sync"
	"testing"

	"github.com/altessa-s/rmlines-go"
)

type collected struct {
	mu     sync.Mutex
	levels []rmlines.LogLevel
	lines  []string
}

func (c *collected) log(level rmlines.LogLevel, message string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.levels = append(c.levels, level)
	c.lines = append(c.lines, message)
}

func (c *collected) saw(level rmlines.LogLevel, needle string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, line := range c.lines {
		if c.levels[i] == level && strings.Contains(line, needle) {
			return true
		}
	}
	return false
}

func (c *collected) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.lines)
}

func TestALostStrokeReachesTheLogger(t *testing.T) {
	var seen collected
	rmlines.SetLogger(seen.log)
	defer rmlines.SetLogger(nil)

	page := mustParse(t, pageWithBlock(0x05, make([]byte, 24)))
	if page.SkippedBlocks != 1 {
		t.Errorf("skipped = %d, want 1", page.SkippedBlocks)
	}
	if !seen.saw(rmlines.LogError, "unreadable") {
		t.Errorf("the loss was not said out loud: %q", seen.lines)
	}
}

func TestNilSilencesTheLogger(t *testing.T) {
	var seen collected
	rmlines.SetLogger(seen.log)
	rmlines.SetLogger(nil)
	mustParse(t, pageWithBlock(0x05, make([]byte, 24)))
	if seen.count() != 0 {
		t.Errorf("a silenced logger heard %q", seen.lines)
	}
}

func TestTheLatestLoggerStands(t *testing.T) {
	var first, second collected
	rmlines.SetLogger(first.log)
	rmlines.SetLogger(second.log)
	defer rmlines.SetLogger(nil)
	mustParse(t, pageWithBlock(0x05, make([]byte, 24)))
	if first.count() != 0 || second.count() == 0 {
		t.Errorf("first heard %d lines, second %d", first.count(), second.count())
	}
}

// The logger may be replaced while concurrent parses wait for their handlers.
func TestTheLoggerUnderConcurrentParsing(t *testing.T) {
	var seen collected
	rmlines.SetLogger(seen.log)
	defer rmlines.SetLogger(nil)
	damaged := pageWithBlock(0x05, make([]byte, 24))

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 20 {
				if _, err := rmlines.Parse(damaged); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range 20 {
			if i%2 == 0 {
				rmlines.SetLogger(nil)
			} else {
				rmlines.SetLogger(seen.log)
			}
		}
	}()
	wg.Wait()
}

// A handler that panics loses its line, not the parse.
func TestAPanickingLoggerDoesNotStopTheParse(t *testing.T) {
	calls := 0
	rmlines.SetLogger(func(rmlines.LogLevel, string) {
		calls++
		panic("a handler that must not panic")
	})
	defer rmlines.SetLogger(nil)
	page, err := rmlines.Parse(pageWithBlock(0x05, make([]byte, 24)))
	if err != nil || page.SkippedBlocks != 1 {
		t.Fatalf("parse = %+v, %v; want one skipped block", page, err)
	}
	if calls == 0 {
		t.Errorf("the handler was never called")
	}
}

// The C library keeps every sink it is given, so the sink is installed once
// however often the logger changes.
func TestTheSinkIsInstalledOnce(t *testing.T) {
	var seen collected
	for range 1000 {
		rmlines.SetLogger(seen.log)
		rmlines.SetLogger(nil)
	}
	if got := rmlines.LogInstalls(); got != 1 {
		t.Errorf("installed %d times, want 1", got)
	}
	rmlines.SetLogger(seen.log)
	defer rmlines.SetLogger(nil)
	mustParse(t, pageWithBlock(0x05, make([]byte, 24)))
	if seen.count() == 0 {
		t.Errorf("the logger set after the cycles hears nothing")
	}
}

func TestLoggerGoexitReturns(t *testing.T) {
	rmlines.SetLogger(func(rmlines.LogLevel, string) { runtime.Goexit() })
	defer rmlines.SetLogger(nil)
	done := make(chan struct{})
	returned := false
	go func() {
		defer close(done)
		page, err := rmlines.Parse(pageWithBlock(0x05, make([]byte, 24)))
		returned = err == nil && page.SkippedBlocks == 1
	}()
	<-done
	if !returned {
		t.Fatal("Parse did not return its normal result")
	}
}

func TestLoggerExitsOnSecondLine(t *testing.T) {
	draft := sampleDraft()
	for i := range draft.Paragraphs {
		draft.Paragraphs[i].Style = ptr[uint8](99)
	}
	data, err := rmlines.Write(draft)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	rmlines.SetLogger(func(rmlines.LogLevel, string) {
		calls++
		if calls == 2 {
			runtime.Goexit()
		}
	})
	defer rmlines.SetLogger(nil)
	done := make(chan struct{})
	returned := false
	go func() {
		defer close(done)
		page, err := rmlines.Parse(data)
		returned = err == nil && page.Text != nil && len(page.Text.Paragraphs) == 3
	}()
	<-done
	if !returned || calls != 3 {
		t.Fatalf("returned = %v, calls = %d, want normal result and three lines", returned, calls)
	}
}

func TestAbnormalLoggerConcurrentParsing(t *testing.T) {
	rmlines.SetLogger(func(rmlines.LogLevel, string) {
		if rand.IntN(2) == 0 {
			panic("logger panic")
		}
		runtime.Goexit()
	})
	defer rmlines.SetLogger(nil)
	data := pageWithBlock(0x05, make([]byte, 24))
	var wg sync.WaitGroup
	for range 128 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 20 {
				page, err := rmlines.Parse(data)
				if err != nil || page.SkippedBlocks != 1 {
					t.Errorf("parse = %+v, %v", page, err)
				}
			}
		}()
	}
	wg.Wait()
}

func TestRepeatedLoggerExit(t *testing.T) {
	rmlines.SetLogger(func(rmlines.LogLevel, string) { runtime.Goexit() })
	defer rmlines.SetLogger(nil)
	data := pageWithBlock(0x05, make([]byte, 24))
	external := func() uint64 {
		runtime.GC()
		samples := []metrics.Sample{{Name: "/memory/classes/total:bytes"}, {Name: "/memory/classes/heap/objects:bytes"}}
		metrics.Read(samples)
		return samples[0].Value.Uint64() - samples[1].Value.Uint64()
	}
	before := external()
	for range 4000 {
		page, err := rmlines.Parse(data)
		if err != nil || page.SkippedBlocks != 1 {
			t.Fatalf("parse = %+v, %v", page, err)
		}
	}
	after := external()
	t.Logf("heap-external bytes: before %d, after %d", before, after)
	if after > before && after-before > 64<<20 {
		t.Fatalf("heap-external growth = %d bytes, want at most 64 MiB", after-before)
	}
}
