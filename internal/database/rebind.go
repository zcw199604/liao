package database

import (
	"strconv"
	"strings"
)

// RebindDollar replaces '?' placeholders with $1, $2, ... for PostgreSQL.
//
// This implementation intentionally ignores '?' inside single-quoted string literals.
// It is not a full SQL parser, but is sufficient for typical hand-written SQL used in this project.
func RebindDollar(query string) string {
	if strings.IndexByte(query, '?') < 0 {
		return query
	}

	var b strings.Builder
	// Worst case: every byte could expand by a few chars; grow a bit to reduce reallocations.
	b.Grow(len(query) + 8)

	argN := 1
	inSingle := false
	for i := 0; i < len(query); i++ {
		ch := query[i]
		if ch == '\'' {
			// Handle escaped single quote inside a string: '' (two single quotes)
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

		if ch == '?' && !inSingle {
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(argN))
			argN++
			continue
		}
		b.WriteByte(ch)
	}
	return b.String()
}
