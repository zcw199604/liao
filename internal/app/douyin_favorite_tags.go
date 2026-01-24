package app

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

var (
	ErrDouyinFavoriteTagAlreadyExists = errors.New("标签已存在")
	ErrDouyinFavoriteTagInvalidMode   = errors.New("标签操作 mode 不合法")
)

type DouyinFavoriteTag struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	SortOrder  int64  `json:"sortOrder"`
	Count      int64  `json:"count"`
	CreateTime string `json:"createTime"`
	UpdateTime string `json:"updateTime"`
}

func isMySQLDuplicateEntry(err error) bool {
	var myErr *mysql.MySQLError
	if errors.As(err, &myErr) && myErr != nil && myErr.Number == 1062 {
		return true
	}
	return false
}

func normalizeStringList(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, raw := range in {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func normalizeInt64List(in []int64) []int64 {
	seen := make(map[int64]struct{}, len(in))
	out := make([]int64, 0, len(in))
	for _, v := range in {
		if v <= 0 {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func (s *DouyinFavoriteService) ListUserTags(ctx context.Context) ([]DouyinFavoriteTag, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.name, t.sort_order, COUNT(m.sec_user_id) AS cnt, t.created_at, t.updated_at
		FROM douyin_favorite_user_tag t
		LEFT JOIN douyin_favorite_user_tag_map m ON m.tag_id = t.id
		GROUP BY t.id, t.name, t.sort_order, t.created_at, t.updated_at
		ORDER BY t.sort_order ASC, t.updated_at DESC, t.id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DouyinFavoriteTag
	for rows.Next() {
		var t DouyinFavoriteTag
		var createdAt, updatedAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.Name, &t.SortOrder, &t.Count, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		t.CreateTime = formatNullLocalDateTimeISO(createdAt)
		t.UpdateTime = formatNullLocalDateTimeISO(updatedAt)
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []DouyinFavoriteTag{}
	}
	return out, nil
}

func (s *DouyinFavoriteService) AddUserTag(ctx context.Context, name string) (*DouyinFavoriteTag, error) {
	var maxSortOrder sql.NullInt64
	if err := s.db.QueryRowContext(ctx, "SELECT MAX(sort_order) FROM douyin_favorite_user_tag").Scan(&maxSortOrder); err != nil {
		return nil, err
	}
	nextSortOrder := int64(0)
	if maxSortOrder.Valid {
		nextSortOrder = maxSortOrder.Int64
	}
	nextSortOrder += 1

	now := time.Now()
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO douyin_favorite_user_tag (name, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`, strings.TrimSpace(name), nextSortOrder, now, now)
	if err != nil {
		if isMySQLDuplicateEntry(err) {
			return nil, ErrDouyinFavoriteTagAlreadyExists
		}
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.findUserTagByID(ctx, id)
}

func (s *DouyinFavoriteService) UpdateUserTag(ctx context.Context, id int64, name string) (*DouyinFavoriteTag, error) {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE douyin_favorite_user_tag
		SET name = ?, updated_at = ?
		WHERE id = ?
	`, strings.TrimSpace(name), now, id)
	if err != nil {
		if isMySQLDuplicateEntry(err) {
			return nil, ErrDouyinFavoriteTagAlreadyExists
		}
		return nil, err
	}
	return s.findUserTagByID(ctx, id)
}

func (s *DouyinFavoriteService) RemoveUserTag(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM douyin_favorite_user_tag_map WHERE tag_id = ?", id); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM douyin_favorite_user_tag WHERE id = ?", id); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *DouyinFavoriteService) findUserTagByID(ctx context.Context, id int64) (*DouyinFavoriteTag, error) {
	var t DouyinFavoriteTag
	var createdAt, updatedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT t.id, t.name, t.sort_order,
		       (SELECT COUNT(*) FROM douyin_favorite_user_tag_map m WHERE m.tag_id = t.id) AS cnt,
		       t.created_at, t.updated_at
		FROM douyin_favorite_user_tag t
		WHERE t.id = ?
		LIMIT 1
	`, id).Scan(&t.ID, &t.Name, &t.SortOrder, &t.Count, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t.CreateTime = formatNullLocalDateTimeISO(createdAt)
	t.UpdateTime = formatNullLocalDateTimeISO(updatedAt)
	return &t, nil
}

func (s *DouyinFavoriteService) ApplyUserTags(ctx context.Context, secUserIDs []string, tagIDs []int64, mode string) error {
	targetIDs := normalizeStringList(secUserIDs)
	tagIDs = normalizeInt64List(tagIDs)
	if len(targetIDs) == 0 {
		return nil
	}

	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode == "" {
		mode = "set"
	}
	if mode != "set" && mode != "add" && mode != "remove" {
		return ErrDouyinFavoriteTagInvalidMode
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	now := time.Now()

	switch mode {
	case "set":
		for _, secUserID := range targetIDs {
			if _, err := tx.ExecContext(ctx, "DELETE FROM douyin_favorite_user_tag_map WHERE sec_user_id = ?", secUserID); err != nil {
				_ = tx.Rollback()
				return err
			}
			for _, tagID := range tagIDs {
				if _, err := tx.ExecContext(ctx, `
					INSERT IGNORE INTO douyin_favorite_user_tag_map (sec_user_id, tag_id, created_at)
					VALUES (?, ?, ?)
				`, secUserID, tagID, now); err != nil {
					_ = tx.Rollback()
					return err
				}
			}
		}
	case "add":
		if len(tagIDs) == 0 {
			return tx.Commit()
		}
		for _, secUserID := range targetIDs {
			for _, tagID := range tagIDs {
				if _, err := tx.ExecContext(ctx, `
					INSERT IGNORE INTO douyin_favorite_user_tag_map (sec_user_id, tag_id, created_at)
					VALUES (?, ?, ?)
				`, secUserID, tagID, now); err != nil {
					_ = tx.Rollback()
					return err
				}
			}
		}
	case "remove":
		if len(tagIDs) == 0 {
			return tx.Commit()
		}
		for _, secUserID := range targetIDs {
			for _, tagID := range tagIDs {
				if _, err := tx.ExecContext(ctx, `
					DELETE FROM douyin_favorite_user_tag_map
					WHERE sec_user_id = ? AND tag_id = ?
				`, secUserID, tagID); err != nil {
					_ = tx.Rollback()
					return err
				}
			}
		}
	}
	return tx.Commit()
}

func (s *DouyinFavoriteService) listUserTagIDsBySecUserID(ctx context.Context, secUserID string) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT tag_id
		FROM douyin_favorite_user_tag_map
		WHERE sec_user_id = ?
		ORDER BY tag_id ASC
	`, secUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *DouyinFavoriteService) fillUserTagIDs(ctx context.Context, items []DouyinFavoriteUser) error {
	if len(items) == 0 {
		return nil
	}
	tagMap, err := s.listAllUserTagIDs(ctx)
	if err != nil {
		return err
	}
	for i := range items {
		tagIDs := tagMap[items[i].SecUserID]
		if tagIDs == nil {
			tagIDs = []int64{}
		}
		items[i].TagIDs = tagIDs
	}
	return nil
}

func (s *DouyinFavoriteService) listAllUserTagIDs(ctx context.Context) (map[string][]int64, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT sec_user_id, tag_id
		FROM douyin_favorite_user_tag_map
		ORDER BY sec_user_id ASC, tag_id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string][]int64)
	for rows.Next() {
		var secUserID string
		var tagID int64
		if err := rows.Scan(&secUserID, &tagID); err != nil {
			return nil, err
		}
		secUserID = strings.TrimSpace(secUserID)
		if secUserID == "" || tagID <= 0 {
			continue
		}
		out[secUserID] = append(out[secUserID], tagID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *DouyinFavoriteService) ReorderUserTags(ctx context.Context, tagIDs []int64) error {
	tagIDs = normalizeInt64List(tagIDs)
	if len(tagIDs) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	for idx, id := range tagIDs {
		if _, err := tx.ExecContext(ctx, "UPDATE douyin_favorite_user_tag SET sort_order = ? WHERE id = ?", int64(idx+1), id); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (s *DouyinFavoriteService) ListAwemeTags(ctx context.Context) ([]DouyinFavoriteTag, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.name, t.sort_order, COUNT(m.aweme_id) AS cnt, t.created_at, t.updated_at
		FROM douyin_favorite_aweme_tag t
		LEFT JOIN douyin_favorite_aweme_tag_map m ON m.tag_id = t.id
		GROUP BY t.id, t.name, t.sort_order, t.created_at, t.updated_at
		ORDER BY t.sort_order ASC, t.updated_at DESC, t.id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DouyinFavoriteTag
	for rows.Next() {
		var t DouyinFavoriteTag
		var createdAt, updatedAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.Name, &t.SortOrder, &t.Count, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		t.CreateTime = formatNullLocalDateTimeISO(createdAt)
		t.UpdateTime = formatNullLocalDateTimeISO(updatedAt)
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []DouyinFavoriteTag{}
	}
	return out, nil
}

func (s *DouyinFavoriteService) AddAwemeTag(ctx context.Context, name string) (*DouyinFavoriteTag, error) {
	var maxSortOrder sql.NullInt64
	if err := s.db.QueryRowContext(ctx, "SELECT MAX(sort_order) FROM douyin_favorite_aweme_tag").Scan(&maxSortOrder); err != nil {
		return nil, err
	}
	nextSortOrder := int64(0)
	if maxSortOrder.Valid {
		nextSortOrder = maxSortOrder.Int64
	}
	nextSortOrder += 1

	now := time.Now()
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO douyin_favorite_aweme_tag (name, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`, strings.TrimSpace(name), nextSortOrder, now, now)
	if err != nil {
		if isMySQLDuplicateEntry(err) {
			return nil, ErrDouyinFavoriteTagAlreadyExists
		}
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.findAwemeTagByID(ctx, id)
}

func (s *DouyinFavoriteService) UpdateAwemeTag(ctx context.Context, id int64, name string) (*DouyinFavoriteTag, error) {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE douyin_favorite_aweme_tag
		SET name = ?, updated_at = ?
		WHERE id = ?
	`, strings.TrimSpace(name), now, id)
	if err != nil {
		if isMySQLDuplicateEntry(err) {
			return nil, ErrDouyinFavoriteTagAlreadyExists
		}
		return nil, err
	}
	return s.findAwemeTagByID(ctx, id)
}

func (s *DouyinFavoriteService) RemoveAwemeTag(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM douyin_favorite_aweme_tag_map WHERE tag_id = ?", id); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM douyin_favorite_aweme_tag WHERE id = ?", id); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *DouyinFavoriteService) findAwemeTagByID(ctx context.Context, id int64) (*DouyinFavoriteTag, error) {
	var t DouyinFavoriteTag
	var createdAt, updatedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT t.id, t.name, t.sort_order,
		       (SELECT COUNT(*) FROM douyin_favorite_aweme_tag_map m WHERE m.tag_id = t.id) AS cnt,
		       t.created_at, t.updated_at
		FROM douyin_favorite_aweme_tag t
		WHERE t.id = ?
		LIMIT 1
	`, id).Scan(&t.ID, &t.Name, &t.SortOrder, &t.Count, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t.CreateTime = formatNullLocalDateTimeISO(createdAt)
	t.UpdateTime = formatNullLocalDateTimeISO(updatedAt)
	return &t, nil
}

func (s *DouyinFavoriteService) ApplyAwemeTags(ctx context.Context, awemeIDs []string, tagIDs []int64, mode string) error {
	targetIDs := normalizeStringList(awemeIDs)
	tagIDs = normalizeInt64List(tagIDs)
	if len(targetIDs) == 0 {
		return nil
	}

	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode == "" {
		mode = "set"
	}
	if mode != "set" && mode != "add" && mode != "remove" {
		return ErrDouyinFavoriteTagInvalidMode
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	now := time.Now()

	switch mode {
	case "set":
		for _, awemeID := range targetIDs {
			if _, err := tx.ExecContext(ctx, "DELETE FROM douyin_favorite_aweme_tag_map WHERE aweme_id = ?", awemeID); err != nil {
				_ = tx.Rollback()
				return err
			}
			for _, tagID := range tagIDs {
				if _, err := tx.ExecContext(ctx, `
					INSERT IGNORE INTO douyin_favorite_aweme_tag_map (aweme_id, tag_id, created_at)
					VALUES (?, ?, ?)
				`, awemeID, tagID, now); err != nil {
					_ = tx.Rollback()
					return err
				}
			}
		}
	case "add":
		if len(tagIDs) == 0 {
			return tx.Commit()
		}
		for _, awemeID := range targetIDs {
			for _, tagID := range tagIDs {
				if _, err := tx.ExecContext(ctx, `
					INSERT IGNORE INTO douyin_favorite_aweme_tag_map (aweme_id, tag_id, created_at)
					VALUES (?, ?, ?)
				`, awemeID, tagID, now); err != nil {
					_ = tx.Rollback()
					return err
				}
			}
		}
	case "remove":
		if len(tagIDs) == 0 {
			return tx.Commit()
		}
		for _, awemeID := range targetIDs {
			for _, tagID := range tagIDs {
				if _, err := tx.ExecContext(ctx, `
					DELETE FROM douyin_favorite_aweme_tag_map
					WHERE aweme_id = ? AND tag_id = ?
				`, awemeID, tagID); err != nil {
					_ = tx.Rollback()
					return err
				}
			}
		}
	}
	return tx.Commit()
}

func (s *DouyinFavoriteService) ReorderAwemeTags(ctx context.Context, tagIDs []int64) error {
	tagIDs = normalizeInt64List(tagIDs)
	if len(tagIDs) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	for idx, id := range tagIDs {
		if _, err := tx.ExecContext(ctx, "UPDATE douyin_favorite_aweme_tag SET sort_order = ? WHERE id = ?", int64(idx+1), id); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (s *DouyinFavoriteService) listAwemeTagIDsByID(ctx context.Context, awemeID string) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT tag_id
		FROM douyin_favorite_aweme_tag_map
		WHERE aweme_id = ?
		ORDER BY tag_id ASC
	`, awemeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *DouyinFavoriteService) fillAwemeTagIDs(ctx context.Context, items []DouyinFavoriteAweme) error {
	if len(items) == 0 {
		return nil
	}
	tagMap, err := s.listAllAwemeTagIDs(ctx)
	if err != nil {
		return err
	}
	for i := range items {
		tagIDs := tagMap[items[i].AwemeID]
		if tagIDs == nil {
			tagIDs = []int64{}
		}
		items[i].TagIDs = tagIDs
	}
	return nil
}

func (s *DouyinFavoriteService) listAllAwemeTagIDs(ctx context.Context) (map[string][]int64, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT aweme_id, tag_id
		FROM douyin_favorite_aweme_tag_map
		ORDER BY aweme_id ASC, tag_id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string][]int64)
	for rows.Next() {
		var awemeID string
		var tagID int64
		if err := rows.Scan(&awemeID, &tagID); err != nil {
			return nil, err
		}
		awemeID = strings.TrimSpace(awemeID)
		if awemeID == "" || tagID <= 0 {
			continue
		}
		out[awemeID] = append(out[awemeID], tagID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
