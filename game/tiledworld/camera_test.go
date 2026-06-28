package main

import (
	"testing"
)

func TestCamera(t *testing.T) {
	orig := pt(0, 0)
	size := pt(5, 5)
	c := NewCamera(orig, size)
	if c.Origin != orig {
		t.Fatalf("camera origin: want %v , got %v", orig, c.Origin)
	}
	if c.Size != size {
		t.Fatalf("camera origin: want %v , got %v", size, c.Size)
	}
	pos := pt(-1, -1)
	c.Follow(pos)
	orig = pt(-1, -1)
	if c.Origin != orig {
		t.Fatalf("camera origin: want %v , got %v", orig, c.Origin)
	}
	// inner rect should contain (5, 5), with margin 0
	pos = pt(5, 5)
	c.Follow(pos)
	orig = pt(1, 1)
	if c.Origin != orig {
		t.Fatalf("camera origin: want %v , got %v", orig, c.Origin)
	}
	c.FollowMargin = 1
	pos = pt(-1, -1)
	c.Follow(pos)
	orig = pt(-2, -2)
	if c.Origin != orig {
		t.Fatalf("camera size: want %v , got %v", size, c.Size)
	}
	// inner rect should contain (5, 5), with margin 1
	pos = pt(5, 5)
	c.Follow(pos)
	orig = pt(2, 2)
	if c.Origin != orig {
		t.Fatalf("camera origin: want %v , got %v", orig, c.Origin)
	}
}
