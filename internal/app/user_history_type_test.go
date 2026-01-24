package app

import "testing"

func TestPickHigherPriorityType(t *testing.T) {
	if got := pickHigherPriorityType("video", ""); got != "video" {
		t.Fatalf("got=%q", got)
	}
	if got := pickHigherPriorityType("", "audio"); got != "audio" {
		t.Fatalf("got=%q", got)
	}
	if got := pickHigherPriorityType("video", "image"); got != "image" {
		t.Fatalf("got=%q", got)
	}
	if got := pickHigherPriorityType("image", "video"); got != "image" {
		t.Fatalf("got=%q", got)
	}
}
