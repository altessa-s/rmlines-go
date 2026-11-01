// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package rmlines_test

import (
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

// The logger is called on whichever goroutine parses, and may be replaced
// while they do.
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
