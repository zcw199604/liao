package database

import (
	"fmt"
	"reflect"
	"strings"
)

// ExpandIn expands slice/array args for SQL "IN (?)" style queries.
//
// Example:
//
//	q, args, _ := ExpandIn("... WHERE id IN (?) AND a=?", []int{1,2,3}, 9)
//	// q  == "... WHERE id IN (?,?,?) AND a=?"
//	// args == []any{1,2,3,9}
//
// Call ExpandIn before Dialect.Rebind (Postgres requires $1 style placeholders).
func ExpandIn(query string, args ...any) (string, []any, error) {
	if len(args) == 0 {
		return query, nil, nil
	}

	var b strings.Builder
	b.Grow(len(query) + 8)

	outArgs := make([]any, 0, len(args))
	argIdx := 0
	inSingle := false

	for i := 0; i < len(query); i++ {
		ch := query[i]
		if ch == '\'' {
			// Same quote handling strategy as RebindDollar.
			if inSingle && i+1 < len(query) && query[i+1] == '\'' {
				b.WriteByte('\'')
				b.WriteByte('\'')
				i++
				continue
			}
			inSingle = !inSingle
			b.WriteByte(ch)
			continue
		}

		if ch != '?' || inSingle {
			b.WriteByte(ch)
			continue
		}

		if argIdx >= len(args) {
			return "", nil, fmt.Errorf("not enough args for query placeholders")
		}

		arg := args[argIdx]
		argIdx++

		if v, ok := expandableSliceValue(arg); ok {
			if v.Len() == 0 {
				return "", nil, fmt.Errorf("empty slice for IN clause")
			}
			for j := 0; j < v.Len(); j++ {
				if j > 0 {
					b.WriteByte(',')
				}
				b.WriteByte('?')
				outArgs = append(outArgs, v.Index(j).Interface())
			}
			continue
		}

		b.WriteByte('?')
		outArgs = append(outArgs, arg)
	}

	if argIdx != len(args) {
		return "", nil, fmt.Errorf("too many args for query placeholders")
	}
	return b.String(), outArgs, nil
}

func expandableSliceValue(arg any) (reflect.Value, bool) {
	v := reflect.ValueOf(arg)
	if !v.IsValid() {
		return reflect.Value{}, false
	}
	kind := v.Kind()
	if kind != reflect.Slice && kind != reflect.Array {
		return reflect.Value{}, false
	}
	// Treat []byte as scalar.
	if v.Type().Elem().Kind() == reflect.Uint8 {
		return reflect.Value{}, false
	}
	return v, true
}
