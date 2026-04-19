package main

import (
	"image"
	"testing"
)

func TestCamera(t *testing.T) {
	orig := image.Pt(0, 0)
	size := image.Pt(5, 5)
	c := NewCamera(orig, size)
	if c.Origin != orig {
		t.Fatalf("camera origin: want %v , got %v", orig, c.Origin)
	}
	if c.Size != size {
		t.Fatalf("camera origin: want %v , got %v", size, c.Size)
	}
	pos := image.Pt(-1, -1)
	c.Follow(pos)
	orig = image.Pt(-1, -1)
	if c.Origin != orig {
		t.Fatalf("camera origin: want %v , got %v", orig, c.Origin)
	}
	// inner rect should contain (5, 5), with margin 0
	pos = image.Pt(5, 5)
	c.Follow(pos)
	orig = image.Pt(1, 1)
	if c.Origin != orig {
		t.Fatalf("camera origin: want %v , got %v", orig, c.Origin)
	}
	c.FollowMargin = 1
	pos = image.Pt(-1, -1)
	c.Follow(pos)
	orig = image.Pt(-2, -2)
	if c.Origin != orig {
		t.Fatalf("camera size: want %v , got %v", size, c.Size)
	}
	// inner rect should contain (5, 5), with margin 1
	pos = image.Pt(5, 5)
	c.Follow(pos)
	orig = image.Pt(2, 2)
	if c.Origin != orig {
		t.Fatalf("camera origin: want %v , got %v", orig, c.Origin)
	}
}
