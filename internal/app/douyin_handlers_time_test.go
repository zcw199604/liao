package app

import (
	"testing"
	"time"
)

func TestFormatUnixTimestampISO(t *testing.T) {
	if got := formatUnixTimestampISO(0); got != "" {
		t.Fatalf("formatUnixTimestampISO(0)=%q, want empty", got)
	}
	if got := formatUnixTimestampISO(-1); got != "" {
		t.Fatalf("formatUnixTimestampISO(-1)=%q, want empty", got)
	}

	sec := int64(1700000000)
	want := formatLocalDateTimeISO(time.Unix(sec, 0).In(time.Local))
	if got := formatUnixTimestampISO(sec); got != want {
		t.Fatalf("formatUnixTimestampISO(sec)=%q, want %q", got, want)
	}
	if got := formatUnixTimestampISO(sec * 1000); got != want {
		t.Fatalf("formatUnixTimestampISO(ms)=%q, want %q", got, want)
	}
}
