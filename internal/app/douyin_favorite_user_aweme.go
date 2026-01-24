package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
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

	unique := make(map[string]DouyinFavoriteUserAwemeUpsert)
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
		unique[awemeID] = it
	}
	if len(unique) == 0 {
		return 0, nil
	}

	awemeIDs := make([]string, 0, len(unique))
	for id := range unique {
		awemeIDs = append(awemeIDs, id)
	}
	sort.Strings(awemeIDs)

	existing := make(map[string]struct{}, len(awemeIDs))
	if len(awemeIDs) > 0 {
		placeholders := make([]string, 0, len(awemeIDs))
		args := make([]any, 0, len(awemeIDs)+1)
		args = append(args, secUserID)
		for _, id := range awemeIDs {
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
	for _, id := range awemeIDs {
		if _, ok := existing[id]; !ok {
			added += 1
		}
	}

	now := time.Now()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}

	for _, id := range awemeIDs {
		it := unique[id]
		var downloadsValue any
		if len(it.Downloads) > 0 {
			if b, err := json.Marshal(it.Downloads); err == nil {
				downloadsValue = nullIfEmpty(string(b))
			}
		}

		_, err := tx.ExecContext(ctx, `
			INSERT INTO douyin_favorite_user_aweme (
				sec_user_id, aweme_id, type, description, cover_url, downloads,
				created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				type = COALESCE(VALUES(type), type),
				description = COALESCE(VALUES(description), description),
				cover_url = COALESCE(VALUES(cover_url), cover_url),
				downloads = COALESCE(VALUES(downloads), downloads),
				updated_at = VALUES(updated_at)
		`,
			secUserID,
			it.AwemeID,
			nullIfEmpty(it.Type),
			nullIfEmpty(it.Desc),
			nullIfEmpty(it.CoverURL),
			downloadsValue,
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
		ORDER BY created_at DESC
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
