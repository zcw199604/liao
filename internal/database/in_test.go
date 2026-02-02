package database

import "testing"

func TestExpandIn_ExpandsSliceArgs(t *testing.T) {
	q, outArgs, err := ExpandIn("SELECT * FROM t WHERE id IN (?) AND a=?", []int{1, 2, 3}, 9)
	if err != nil {
		t.Fatalf("ExpandIn: %v", err)
	}
	if q != "SELECT * FROM t WHERE id IN (?,?,?) AND a=?" {
		t.Fatalf("q=%q", q)
	}
	if len(outArgs) != 4 {
		t.Fatalf("args=%v", outArgs)
	}
	if outArgs[0].(int) != 1 || outArgs[1].(int) != 2 || outArgs[2].(int) != 3 || outArgs[3].(int) != 9 {
		t.Fatalf("args=%v", outArgs)
	}
}

func TestExpandIn_RejectsEmptySlice(t *testing.T) {
	if _, _, err := ExpandIn("SELECT * FROM t WHERE id IN (?)", []int{}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestExpandIn_DoesNotExpandBytes(t *testing.T) {
	q, outArgs, err := ExpandIn("SELECT ?::blob", []byte{1, 2, 3})
	if err != nil {
		t.Fatalf("ExpandIn: %v", err)
	}
	if q != "SELECT ?::blob" {
		t.Fatalf("q=%q", q)
	}
	if len(outArgs) != 1 {
		t.Fatalf("args=%v", outArgs)
	}
	if got := outArgs[0].([]byte); len(got) != 3 {
		t.Fatalf("args=%v", outArgs)
	}
}
