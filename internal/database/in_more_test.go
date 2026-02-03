package database

import "testing"

func TestExpandIn_NoArgsReturnsQueryAndNilArgs(t *testing.T) {
	q, outArgs, err := ExpandIn("SELECT 1")
	if err != nil {
		t.Fatalf("ExpandIn: %v", err)
	}
	if q != "SELECT 1" {
		t.Fatalf("q=%q", q)
	}
	if outArgs != nil {
		t.Fatalf("args=%v, want nil", outArgs)
	}
}

func TestExpandIn_NotEnoughArgs(t *testing.T) {
	if _, _, err := ExpandIn("SELECT ? AND ?", 1); err == nil {
		t.Fatalf("expected error")
	}
}

func TestExpandIn_TooManyArgs(t *testing.T) {
	if _, _, err := ExpandIn("SELECT ?", 1, 2); err == nil {
		t.Fatalf("expected error")
	}
}

func TestExpandIn_ExpandsArrayArgs(t *testing.T) {
	q, outArgs, err := ExpandIn("WHERE id IN (?)", [3]int{1, 2, 3})
	if err != nil {
		t.Fatalf("ExpandIn: %v", err)
	}
	if q != "WHERE id IN (?,?,?)" {
		t.Fatalf("q=%q", q)
	}
	if len(outArgs) != 3 || outArgs[0].(int) != 1 || outArgs[1].(int) != 2 || outArgs[2].(int) != 3 {
		t.Fatalf("args=%v", outArgs)
	}
}

func TestExpandIn_IgnoresPlaceholdersInSingleQuotes(t *testing.T) {
	q, outArgs, err := ExpandIn("SELECT '?' AS lit, id IN (?)", []int{1, 2})
	if err != nil {
		t.Fatalf("ExpandIn: %v", err)
	}
	if q != "SELECT '?' AS lit, id IN (?,?)" {
		t.Fatalf("q=%q", q)
	}
	if len(outArgs) != 2 || outArgs[0].(int) != 1 || outArgs[1].(int) != 2 {
		t.Fatalf("args=%v", outArgs)
	}
}

func TestExpandIn_HandlesEscapedQuotesInStrings(t *testing.T) {
	q, outArgs, err := ExpandIn("SELECT 'it''s ?' AS lit, id IN (?)", []int{7, 8})
	if err != nil {
		t.Fatalf("ExpandIn: %v", err)
	}
	if q != "SELECT 'it''s ?' AS lit, id IN (?,?)" {
		t.Fatalf("q=%q", q)
	}
	if len(outArgs) != 2 || outArgs[0].(int) != 7 || outArgs[1].(int) != 8 {
		t.Fatalf("args=%v", outArgs)
	}
}

func TestExpandIn_AllowsNilScalarArg(t *testing.T) {
	q, outArgs, err := ExpandIn("SELECT ?", nil)
	if err != nil {
		t.Fatalf("ExpandIn: %v", err)
	}
	if q != "SELECT ?" {
		t.Fatalf("q=%q", q)
	}
	if len(outArgs) != 1 || outArgs[0] != nil {
		t.Fatalf("args=%v", outArgs)
	}
}

