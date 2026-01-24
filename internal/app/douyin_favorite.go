package app

import (
	"context"
	"database/sql"
	"time"
)

type DouyinFavoriteUser struct {
	SecUserID       string `json:"secUserId"`
	SourceInput     string `json:"sourceInput,omitempty"`
	DisplayName     string `json:"displayName,omitempty"`
	AvatarURL       string `json:"avatarUrl,omitempty"`
	ProfileURL      string `json:"profileUrl,omitempty"`
	LastParsedAt    string `json:"lastParsedAt,omitempty"`
	LastParsedCount int    `json:"lastParsedCount,omitempty"`
	CreateTime      string `json:"createTime"`
	UpdateTime      string `json:"updateTime"`
}

type DouyinFavoriteAweme struct {
	AwemeID    string `json:"awemeId"`
	SecUserID  string `json:"secUserId,omitempty"`
	Type       string `json:"type,omitempty"`
	Desc       string `json:"desc,omitempty"`
	CoverURL   string `json:"coverUrl,omitempty"`
	CreateTime string `json:"createTime"`
	UpdateTime string `json:"updateTime"`
}

type DouyinFavoriteService struct {
	db *sql.DB
}

func NewDouyinFavoriteService(db *sql.DB) *DouyinFavoriteService {
	return &DouyinFavoriteService{db: db}
}

type DouyinFavoriteUserUpsert struct {
	SecUserID       string
	SourceInput     string
	DisplayName     string
	AvatarURL       string
	ProfileURL      string
	LastParsedCount *int
	LastParsedRaw   string
}

func (s *DouyinFavoriteService) UpsertUser(ctx context.Context, in DouyinFavoriteUserUpsert) (*DouyinFavoriteUser, error) {
	now := time.Now()

	var countValue any
	if in.LastParsedCount != nil {
		countValue = *in.LastParsedCount
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO douyin_favorite_user (
			sec_user_id, source_input, display_name, avatar_url, profile_url,
			last_parsed_at, last_parsed_count, last_parsed_raw,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			source_input = COALESCE(VALUES(source_input), source_input),
			display_name = COALESCE(VALUES(display_name), display_name),
			avatar_url = COALESCE(VALUES(avatar_url), avatar_url),
			profile_url = COALESCE(VALUES(profile_url), profile_url),
			last_parsed_at = VALUES(last_parsed_at),
			last_parsed_count = COALESCE(VALUES(last_parsed_count), last_parsed_count),
			last_parsed_raw = COALESCE(VALUES(last_parsed_raw), last_parsed_raw),
			updated_at = VALUES(updated_at)
	`,
		in.SecUserID,
		nullIfEmpty(in.SourceInput),
		nullIfEmpty(in.DisplayName),
		nullIfEmpty(in.AvatarURL),
		nullIfEmpty(in.ProfileURL),
		now,
		countValue,
		nullIfEmpty(in.LastParsedRaw),
		now,
		now,
	)
	if err != nil {
		return nil, err
	}

	return s.findUserBySecUserID(ctx, in.SecUserID)
}

func (s *DouyinFavoriteService) RemoveUser(ctx context.Context, secUserID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM douyin_favorite_user WHERE sec_user_id = ?", secUserID)
	return err
}

func (s *DouyinFavoriteService) ListUsers(ctx context.Context) ([]DouyinFavoriteUser, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT sec_user_id, source_input, display_name, avatar_url, profile_url,
		       last_parsed_at, last_parsed_count, created_at, updated_at
		FROM douyin_favorite_user
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DouyinFavoriteUser
	for rows.Next() {
		var u DouyinFavoriteUser
		var sourceInput, displayName, avatarURL, profileURL sql.NullString
		var lastParsedAt sql.NullTime
		var lastParsedCount sql.NullInt64
		var createdAt, updatedAt sql.NullTime

		if err := rows.Scan(
			&u.SecUserID,
			&sourceInput,
			&displayName,
			&avatarURL,
			&profileURL,
			&lastParsedAt,
			&lastParsedCount,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, err
		}

		if sourceInput.Valid {
			u.SourceInput = sourceInput.String
		}
		if displayName.Valid {
			u.DisplayName = displayName.String
		}
		if avatarURL.Valid {
			u.AvatarURL = avatarURL.String
		}
		if profileURL.Valid {
			u.ProfileURL = profileURL.String
		}
		u.LastParsedAt = formatNullLocalDateTimeISO(lastParsedAt)
		if lastParsedCount.Valid {
			u.LastParsedCount = int(lastParsedCount.Int64)
		}
		u.CreateTime = formatNullLocalDateTimeISO(createdAt)
		u.UpdateTime = formatNullLocalDateTimeISO(updatedAt)

		out = append(out, u)
	}

	return out, rows.Err()
}

func (s *DouyinFavoriteService) findUserBySecUserID(ctx context.Context, secUserID string) (*DouyinFavoriteUser, error) {
	var u DouyinFavoriteUser
	var sourceInput, displayName, avatarURL, profileURL sql.NullString
	var lastParsedAt sql.NullTime
	var lastParsedCount sql.NullInt64
	var createdAt, updatedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT sec_user_id, source_input, display_name, avatar_url, profile_url,
		       last_parsed_at, last_parsed_count, created_at, updated_at
		FROM douyin_favorite_user
		WHERE sec_user_id = ?
		LIMIT 1
	`, secUserID).Scan(
		&u.SecUserID,
		&sourceInput,
		&displayName,
		&avatarURL,
		&profileURL,
		&lastParsedAt,
		&lastParsedCount,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if sourceInput.Valid {
		u.SourceInput = sourceInput.String
	}
	if displayName.Valid {
		u.DisplayName = displayName.String
	}
	if avatarURL.Valid {
		u.AvatarURL = avatarURL.String
	}
	if profileURL.Valid {
		u.ProfileURL = profileURL.String
	}
	u.LastParsedAt = formatNullLocalDateTimeISO(lastParsedAt)
	if lastParsedCount.Valid {
		u.LastParsedCount = int(lastParsedCount.Int64)
	}
	u.CreateTime = formatNullLocalDateTimeISO(createdAt)
	u.UpdateTime = formatNullLocalDateTimeISO(updatedAt)

	return &u, nil
}

type DouyinFavoriteAwemeUpsert struct {
	AwemeID   string
	SecUserID string
	Type      string
	Desc      string
	CoverURL  string
	RawDetail string
}

func (s *DouyinFavoriteService) UpsertAweme(ctx context.Context, in DouyinFavoriteAwemeUpsert) (*DouyinFavoriteAweme, error) {
	now := time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO douyin_favorite_aweme (
			aweme_id, sec_user_id, type, description, cover_url, raw_detail,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			sec_user_id = COALESCE(VALUES(sec_user_id), sec_user_id),
			type = COALESCE(VALUES(type), type),
			description = COALESCE(VALUES(description), description),
			cover_url = COALESCE(VALUES(cover_url), cover_url),
			raw_detail = COALESCE(VALUES(raw_detail), raw_detail),
			updated_at = VALUES(updated_at)
	`,
		in.AwemeID,
		nullIfEmpty(in.SecUserID),
		nullIfEmpty(in.Type),
		nullIfEmpty(in.Desc),
		nullIfEmpty(in.CoverURL),
		nullIfEmpty(in.RawDetail),
		now,
		now,
	)
	if err != nil {
		return nil, err
	}

	return s.findAwemeByID(ctx, in.AwemeID)
}

func (s *DouyinFavoriteService) RemoveAweme(ctx context.Context, awemeID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM douyin_favorite_aweme WHERE aweme_id = ?", awemeID)
	return err
}

func (s *DouyinFavoriteService) ListAwemes(ctx context.Context) ([]DouyinFavoriteAweme, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT aweme_id, sec_user_id, type, description, cover_url, created_at, updated_at
		FROM douyin_favorite_aweme
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DouyinFavoriteAweme
	for rows.Next() {
		var it DouyinFavoriteAweme
		var secUserID, typeValue, descValue, coverURL sql.NullString
		var createdAt, updatedAt sql.NullTime

		if err := rows.Scan(&it.AwemeID, &secUserID, &typeValue, &descValue, &coverURL, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		if secUserID.Valid {
			it.SecUserID = secUserID.String
		}
		if typeValue.Valid {
			it.Type = typeValue.String
		}
		if descValue.Valid {
			it.Desc = descValue.String
		}
		if coverURL.Valid {
			it.CoverURL = coverURL.String
		}
		it.CreateTime = formatNullLocalDateTimeISO(createdAt)
		it.UpdateTime = formatNullLocalDateTimeISO(updatedAt)

		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *DouyinFavoriteService) findAwemeByID(ctx context.Context, awemeID string) (*DouyinFavoriteAweme, error) {
	var it DouyinFavoriteAweme
	var secUserID, typeValue, descValue, coverURL sql.NullString
	var createdAt, updatedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT aweme_id, sec_user_id, type, description, cover_url, created_at, updated_at
		FROM douyin_favorite_aweme
		WHERE aweme_id = ?
		LIMIT 1
	`, awemeID).Scan(&it.AwemeID, &secUserID, &typeValue, &descValue, &coverURL, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if secUserID.Valid {
		it.SecUserID = secUserID.String
	}
	if typeValue.Valid {
		it.Type = typeValue.String
	}
	if descValue.Valid {
		it.Desc = descValue.String
	}
	if coverURL.Valid {
		it.CoverURL = coverURL.String
	}
	it.CreateTime = formatNullLocalDateTimeISO(createdAt)
	it.UpdateTime = formatNullLocalDateTimeISO(updatedAt)

	return &it, nil
}
