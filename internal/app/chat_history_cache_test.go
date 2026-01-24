package app

import "testing"

func TestExtractHistoryMessageTid(t *testing.T) {
	cases := []struct {
		name string
		msg  map[string]any
		want string
	}{
		{name: "nil", msg: nil, want: ""},
		{name: "Tid", msg: map[string]any{"Tid": " 123 "}, want: "123"},
		{name: "tid", msg: map[string]any{"tid": " 456 "}, want: "456"},
		{name: "prefer Tid", msg: map[string]any{"Tid": " 1 ", "tid": " 2 "}, want: "1"},
		{name: "ignore nil Tid use tid", msg: map[string]any{"Tid": nil, "tid": " 2 "}, want: "2"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractHistoryMessageTid(tc.msg); got != tc.want {
				t.Fatalf("got=%q want=%q", got, tc.want)
			}
		})
	}
}

func TestExtractHistoryMessageDedupKey(t *testing.T) {
	cases := []struct {
		name string
		msg  map[string]any
		want string
	}{
		{name: "nil", msg: nil, want: ""},
		{name: "tid", msg: map[string]any{"Tid": " 9 "}, want: "tid:9"},
		{
			name: "fallback",
			msg: map[string]any{
				"id":      " a ",
				"toid":    " b ",
				"content": " c ",
				"time":    " t ",
			},
			want: "fallback:a|b|t|c",
		},
		{name: "empty fallback fields", msg: map[string]any{"id": " ", "toid": "", "content": " ", "time": ""}, want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractHistoryMessageDedupKey(tc.msg); got != tc.want {
				t.Fatalf("got=%q want=%q", got, tc.want)
			}
		})
	}
}

func TestMergeHistoryMessages(t *testing.T) {
	t.Run("dedup and include empty keys", func(t *testing.T) {
		primary := []map[string]any{
			{"Tid": "1", "content": "a"},
			{"Tid": "1", "content": "dup"},
			nil,
			{},
			{},
		}
		secondary := []map[string]any{
			{"Tid": "2", "content": "b"},
			{"Tid": "1", "content": "dup2"},
		}

		out := mergeHistoryMessages(primary, secondary, -1)
		if len(out) != 4 {
			t.Fatalf("len=%d want=%d", len(out), 4)
		}
		if extractHistoryMessageTid(out[0]) != "1" {
			t.Fatalf("out[0] tid=%q", extractHistoryMessageTid(out[0]))
		}
		if extractHistoryMessageTid(out[1]) != "" || len(out[1]) != 0 {
			t.Fatalf("out[1]=%v", out[1])
		}
		if extractHistoryMessageTid(out[2]) != "" || len(out[2]) != 0 {
			t.Fatalf("out[2]=%v", out[2])
		}
		if extractHistoryMessageTid(out[3]) != "2" {
			t.Fatalf("out[3] tid=%q", extractHistoryMessageTid(out[3]))
		}
	})

	t.Run("limit", func(t *testing.T) {
		primary := []map[string]any{{"Tid": "1"}, {"Tid": "2"}}
		secondary := []map[string]any{{"Tid": "3"}}

		out := mergeHistoryMessages(primary, secondary, 2)
		if len(out) != 2 {
			t.Fatalf("len=%d want=%d", len(out), 2)
		}
	})
}
