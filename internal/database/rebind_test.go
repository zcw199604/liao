package database

import "testing"

func TestRebindDollar_ReplacesPlaceholders(t *testing.T) {
	in := "SELECT * FROM t WHERE a=? AND b=? AND c=?"
	want := "SELECT * FROM t WHERE a=$1 AND b=$2 AND c=$3"
	if got := RebindDollar(in); got != want {
		t.Fatalf("got=%q want=%q", got, want)
	}
}

func TestRebindDollar_IgnoresQuestionMarkInSingleQuotes(t *testing.T) {
	in := "SELECT '?' AS q, a=? FROM t WHERE b='it''s ?' AND c=?"
	want := "SELECT '?' AS q, a=$1 FROM t WHERE b='it''s ?' AND c=$2"
	if got := RebindDollar(in); got != want {
		t.Fatalf("got=%q want=%q", got, want)
	}
}
