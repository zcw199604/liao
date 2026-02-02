package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"liao/internal/database"
)

type DouyinFavoriteUserAwemeUpsert struct {
	AwemeID   string
	Type      string
	Desc      string
	CoverURL  string
	Downloads []string
}

type DouyinFavoriteUserAwemeRow struct {
	AwemeID    string
	Type       string
	Desc       string
	CoverURL   string
	Downloads  []string
	CreateTime string
	UpdateTime string
}

func parseJSONStringArray(raw sql.NullString) []string {
	if !raw.Valid {
		return nil
	}
	text := strings.TrimSpace(raw.String)
	if text == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil
	}
	return normalizeStringList(out)
}

func (s *DouyinFavoriteService) UpsertUserAwemes(ctx context.Context, secUserID string, items []DouyinFavoriteUserAwemeUpsert) (int, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("db not initialized")
	}
	secUserID = strings.TrimSpace(secUserID)
	if secUserID == "" || len(items) == 0 {
		return 0, nil
	}

	// Preserve the incoming order (usually: newest -> oldest), because this order is used for
	// display in favorite user works list. We still dedupe by aweme_id, but keep the first
	// occurrence order; later duplicates only update the stored metadata.
	unique := make(map[string]DouyinFavoriteUserAwemeUpsert, len(items))
	orderedIDs := make([]string, 0, len(items))
	for _, it := range items {
		awemeID := strings.TrimSpace(it.AwemeID)
		if awemeID == "" {
			continue
		}
		it.AwemeID = awemeID
		it.Type = strings.TrimSpace(it.Type)
		it.Desc = strings.TrimSpace(it.Desc)
		it.CoverURL = strings.TrimSpace(it.CoverURL)
		it.Downloads = normalizeStringList(it.Downloads)
		if _, ok := unique[awemeID]; !ok {
			orderedIDs = append(orderedIDs, awemeID)
		}
		unique[awemeID] = it
	}
	if len(orderedIDs) == 0 {
		return 0, nil
	}

	existing := make(map[string]struct{}, len(orderedIDs))
	if len(orderedIDs) > 0 {
		placeholders := make([]string, 0, len(orderedIDs))
		args := make([]any, 0, len(orderedIDs)+1)
		args = append(args, secUserID)
		for _, id := range orderedIDs {
			placeholders = append(placeholders, "?")
			args = append(args, id)
		}

		query := fmt.Sprintf(`
			SELECT aweme_id
			FROM douyin_favorite_user_aweme
			WHERE sec_user_id = ?
			  AND aweme_id IN (%s)
		`, strings.Join(placeholders, ","))
		rows, err := s.db.QueryContext(ctx, query, args...)
		if err != nil {
			return 0, err
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				_ = rows.Close()
				return 0, err
			}
			id = strings.TrimSpace(id)
			if id != "" {
				existing[id] = struct{}{}
			}
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return 0, err
		}
		_ = rows.Close()
	}

	added := 0
	for _, id := range orderedIDs {
		if _, ok := existing[id]; !ok {
			added += 1
		}
	}

	// Always treat the incoming list as the "front segment" of the desired display order
	// (newest -> oldest as returned by upstream). To keep the display stable across multiple
	// incremental upserts (e.g. pullLatest), we assign this segment to a new sort_order range
	// strictly before the current minimum sort_order, so we never need to renumber the rest.
	var minSortOrder sql.NullInt64
	if err := s.db.QueryRowContext(ctx, `
		SELECT MIN(sort_order)
		FROM douyin_favorite_user_aweme
		WHERE sec_user_id = ?
	`, secUserID).Scan(&minSortOrder); err != nil {
		return 0, err
	}
	baseSortOrder := int64(0)
	if minSortOrder.Valid {
		baseSortOrder = minSortOrder.Int64 - int64(len(orderedIDs))
	}

	now := time.Now()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}

	insertCols := []string{
		"sec_user_id",
		"aweme_id",
		"type",
		"description",
		"cover_url",
		"downloads",
		"sort_order",
		"created_at",
		"updated_at",
	}
	conflictCols := []string{"sec_user_id", "aweme_id"}
	updateCols := []string{"sort_order", "updated_at"}
	updateCoalesceCols := []string{"type", "description", "cover_url", "downloads"}

	for idx, id := range orderedIDs {
		it := unique[id]
		var downloadsValue any
		if len(it.Downloads) > 0 {
			if b, err := json.Marshal(it.Downloads); err == nil {
				downloadsValue = nullIfEmpty(string(b))
			}
		}

		_, err := database.ExecUpsert(
			ctx,
			tx,
			"douyin_favorite_user_aweme",
			insertCols,
			conflictCols,
			updateCols,
			updateCoalesceCols,
			secUserID,
			it.AwemeID,
			nullIfEmpty(it.Type),
			nullIfEmpty(it.Desc),
			nullIfEmpty(it.CoverURL),
			downloadsValue,
			baseSortOrder+int64(idx),
			now,
			now,
		)
		if err != nil {
			_ = tx.Rollback()
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return added, nil
}

func (s *DouyinFavoriteService) ListUserAwemes(ctx context.Context, secUserID string, cursor, count int) ([]DouyinFavoriteUserAwemeRow, int, bool, error) {
	if s == nil || s.db == nil {
		return nil, 0, false, fmt.Errorf("db not initialized")
	}
	secUserID = strings.TrimSpace(secUserID)
	if secUserID == "" {
		return nil, 0, false, nil
	}
	if cursor < 0 {
		cursor = 0
	}
	if count <= 0 {
		count = 20
	}
	if count > 50 {
		count = 50
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT aweme_id, type, description, cover_url, downloads, created_at, updated_at
		FROM douyin_favorite_user_aweme
		WHERE sec_user_id = ?
		ORDER BY sort_order ASC, aweme_id DESC
		LIMIT ? OFFSET ?
	`, secUserID, count+1, cursor)
	if err != nil {
		return nil, 0, false, err
	}
	defer rows.Close()

	out := make([]DouyinFavoriteUserAwemeRow, 0, count+1)
	for rows.Next() {
		var row DouyinFavoriteUserAwemeRow
		var typeValue, descValue, coverURL, downloadsRaw sql.NullString
		var createdAt, updatedAt sql.NullTime
		if err := rows.Scan(&row.AwemeID, &typeValue, &descValue, &coverURL, &downloadsRaw, &createdAt, &updatedAt); err != nil {
			return nil, 0, false, err
		}
		row.AwemeID = strings.TrimSpace(row.AwemeID)
		if typeValue.Valid {
			row.Type = strings.TrimSpace(typeValue.String)
		}
		if descValue.Valid {
			row.Desc = strings.TrimSpace(descValue.String)
		}
		if coverURL.Valid {
			row.CoverURL = strings.TrimSpace(coverURL.String)
		}
		row.Downloads = parseJSONStringArray(downloadsRaw)
		row.CreateTime = formatNullLocalDateTimeISO(createdAt)
		row.UpdateTime = formatNullLocalDateTimeISO(updatedAt)
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, false, err
	}

	hasMore := false
	if len(out) > count {
		hasMore = true
		out = out[:count]
	}
	nextCursor := cursor + len(out)
	return out, nextCursor, hasMore, nil
}
