package app

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

type Favorite struct {
	ID            int64  `json:"id"`
	IdentityID    string `json:"identityId"`
	TargetUserID  string `json:"targetUserId"`
	TargetUserName string `json:"targetUserName,omitempty"`
	CreateTime    string `json:"createTime"`
}

type FavoriteService struct {
	db *sql.DB
}

func NewFavoriteService(db *sql.DB) *FavoriteService {
	return &FavoriteService{db: db}
}

func (s *FavoriteService) Add(ctx context.Context, identityID, targetUserID, targetUserName string) (*Favorite, error) {
	existing, err := s.findByIdentityAndTarget(ctx, identityID, targetUserID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	now := time.Now()
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO chat_favorites (identity_id, target_user_id, target_user_name, create_time) VALUES (?, ?, ?, ?)",
		identityID, targetUserID, nullIfEmpty(targetUserName), now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()

	return &Favorite{
		ID:             id,
		IdentityID:     identityID,
		TargetUserID:   targetUserID,
		TargetUserName: targetUserName,
		CreateTime:     formatLocalDateTimeISO(now),
	}, nil
}

func (s *FavoriteService) Remove(ctx context.Context, identityID, targetUserID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM chat_favorites WHERE identity_id = ? AND target_user_id = ?", identityID, targetUserID)
	return err
}

func (s *FavoriteService) RemoveByID(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM chat_favorites WHERE id = ?", id)
	return err
}

func (s *FavoriteService) ListAll(ctx context.Context) ([]Favorite, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, identity_id, target_user_id, target_user_name, create_time FROM chat_favorites ORDER BY create_time DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Favorite
	for rows.Next() {
		var fav Favorite
		var createTime sql.NullTime
		var targetName sql.NullString
		if err := rows.Scan(&fav.ID, &fav.IdentityID, &fav.TargetUserID, &targetName, &createTime); err != nil {
			return nil, err
		}
		if targetName.Valid {
			fav.TargetUserName = targetName.String
		}
		fav.CreateTime = formatNullLocalDateTimeISO(createTime)
		out = append(out, fav)
	}
	return out, rows.Err()
}

func (s *FavoriteService) IsFavorite(ctx context.Context, identityID, targetUserID string) (bool, error) {
	row := s.db.QueryRowContext(ctx, "SELECT 1 FROM chat_favorites WHERE identity_id = ? AND target_user_id = ? LIMIT 1", identityID, targetUserID)
	var one int
	err := row.Scan(&one)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}
	return false, err
}

func (s *FavoriteService) findByIdentityAndTarget(ctx context.Context, identityID, targetUserID string) (*Favorite, error) {
	var fav Favorite
	var createTime sql.NullTime
	var targetName sql.NullString
	err := s.db.QueryRowContext(ctx, "SELECT id, identity_id, target_user_id, target_user_name, create_time FROM chat_favorites WHERE identity_id = ? AND target_user_id = ? LIMIT 1", identityID, targetUserID).
		Scan(&fav.ID, &fav.IdentityID, &fav.TargetUserID, &targetName, &createTime)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if targetName.Valid {
		fav.TargetUserName = targetName.String
	}
	fav.CreateTime = formatNullLocalDateTimeISO(createTime)
	return &fav, nil
}

func formatLocalDateTimeISO(t time.Time) string {
	return t.Format("2006-01-02T15:04:05")
}

func formatNullLocalDateTimeISO(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return formatLocalDateTimeISO(t.Time)
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
