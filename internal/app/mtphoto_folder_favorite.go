package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"liao/internal/database"
)

const (
	mtPhotoFolderFavoriteMaxTags      = 20
	mtPhotoFolderFavoriteMaxTagLength = 32
	mtPhotoFolderFavoriteMaxNoteRunes = 500
)

type MtPhotoFolderFavorite struct {
	ID         int64    `json:"id"`
	FolderID   int64    `json:"folderId"`
	FolderName string   `json:"folderName"`
	FolderPath string   `json:"folderPath"`
	CoverMD5   string   `json:"coverMd5,omitempty"`
	Tags       []string `json:"tags"`
	Note       string   `json:"note"`
	CreateTime string   `json:"createTime"`
	UpdateTime string   `json:"updateTime"`
}

type MtPhotoFolderFavoriteUpsertInput struct {
	FolderID   int64    `json:"folderId"`
	FolderName string   `json:"folderName"`
	FolderPath string   `json:"folderPath"`
	CoverMD5   string   `json:"coverMd5"`
	Tags       []string `json:"tags"`
	Note       string   `json:"note"`
}

type MtPhotoFolderFavoriteService struct {
	db *database.DB
}

// NewMtPhotoFolderFavoriteService 创建 mtPhoto 文件夹收藏服务。
func NewMtPhotoFolderFavoriteService(db *database.DB) *MtPhotoFolderFavoriteService {
	return &MtPhotoFolderFavoriteService{db: db}
}

func normalizeMtPhotoFolderFavoriteTags(tags []string) ([]string, error) {
	normalized := normalizeStringList(tags)
	if len(normalized) > mtPhotoFolderFavoriteMaxTags {
		return nil, fmt.Errorf("标签数量不能超过 %d", mtPhotoFolderFavoriteMaxTags)
	}
	for _, tag := range normalized {
		if utf8.RuneCountInString(tag) > mtPhotoFolderFavoriteMaxTagLength {
			return nil, fmt.Errorf("单个标签长度不能超过 %d", mtPhotoFolderFavoriteMaxTagLength)
		}
	}
	return normalized, nil
}

func parseMtPhotoFolderFavoriteTags(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}
	var tags []string
	if err := json.Unmarshal([]byte(raw), &tags); err != nil {
		return []string{}
	}
	normalized := normalizeStringList(tags)
	if normalized == nil {
		return []string{}
	}
	return normalized
}

func (s *MtPhotoFolderFavoriteService) validateUpsertInput(in MtPhotoFolderFavoriteUpsertInput) ([]string, string, error) {
	if s == nil || s.db == nil {
		return nil, "", fmt.Errorf("db not initialized")
	}
	if in.FolderID <= 0 {
		return nil, "", fmt.Errorf("folderId 参数非法")
	}

	folderName := strings.TrimSpace(in.FolderName)
	if folderName == "" {
		return nil, "", fmt.Errorf("folderName 不能为空")
	}

	folderPath := strings.TrimSpace(in.FolderPath)
	if folderPath == "" {
		return nil, "", fmt.Errorf("folderPath 不能为空")
	}

	coverMD5 := strings.TrimSpace(in.CoverMD5)
	if coverMD5 != "" && !isValidMD5Hex(coverMD5) {
		return nil, "", fmt.Errorf("coverMd5 参数非法")
	}

	tags, err := normalizeMtPhotoFolderFavoriteTags(in.Tags)
	if err != nil {
		return nil, "", err
	}

	note := strings.TrimSpace(in.Note)
	if utf8.RuneCountInString(note) > mtPhotoFolderFavoriteMaxNoteRunes {
		return nil, "", fmt.Errorf("note 长度不能超过 %d", mtPhotoFolderFavoriteMaxNoteRunes)
	}

	return tags, note, nil
}

// List 按更新时间倒序返回文件夹收藏列表。
func (s *MtPhotoFolderFavoriteService) List(ctx context.Context) ([]MtPhotoFolderFavorite, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db not initialized")
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, folder_id, folder_name, folder_path, cover_md5, tags_json, note, created_at, updated_at
		FROM mtphoto_folder_favorite
		ORDER BY updated_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]MtPhotoFolderFavorite, 0)
	for rows.Next() {
		var item MtPhotoFolderFavorite
		var coverMD5, tagsJSON, note sql.NullString
		var createdAt, updatedAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.FolderID,
			&item.FolderName,
			&item.FolderPath,
			&coverMD5,
			&tagsJSON,
			&note,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, err
		}
		if coverMD5.Valid {
			item.CoverMD5 = strings.TrimSpace(coverMD5.String)
		}
		if tagsJSON.Valid {
			item.Tags = parseMtPhotoFolderFavoriteTags(tagsJSON.String)
		} else {
			item.Tags = []string{}
		}
		if note.Valid {
			item.Note = strings.TrimSpace(note.String)
		}
		item.FolderName = strings.TrimSpace(item.FolderName)
		item.FolderPath = strings.TrimSpace(item.FolderPath)
		item.CreateTime = formatNullLocalDateTimeISO(createdAt)
		item.UpdateTime = formatNullLocalDateTimeISO(updatedAt)
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// Upsert 新增或更新一个文件夹收藏（按 folderId 唯一）。
func (s *MtPhotoFolderFavoriteService) Upsert(ctx context.Context, in MtPhotoFolderFavoriteUpsertInput) (*MtPhotoFolderFavorite, error) {
	tags, note, err := s.validateUpsertInput(in)
	if err != nil {
		return nil, err
	}

	tagsJSONBytes, _ := json.Marshal(tags)
	tagsJSON := string(tagsJSONBytes)
	now := time.Now()

	_, err = database.ExecUpsert(
		ctx,
		s.db,
		"mtphoto_folder_favorite",
		[]string{"folder_id", "folder_name", "folder_path", "cover_md5", "tags_json", "note", "created_at", "updated_at"},
		[]string{"folder_id"},
		[]string{"folder_name", "folder_path", "cover_md5", "tags_json", "note", "updated_at"},
		nil,
		in.FolderID,
		strings.TrimSpace(in.FolderName),
		strings.TrimSpace(in.FolderPath),
		nullIfEmpty(strings.TrimSpace(in.CoverMD5)),
		tagsJSON,
		nullIfEmpty(note),
		now,
		now,
	)
	if err != nil {
		return nil, err
	}

	return s.findByFolderID(ctx, in.FolderID)
}

// Remove 按 folderId 取消收藏。
func (s *MtPhotoFolderFavoriteService) Remove(ctx context.Context, folderID int64) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("db not initialized")
	}
	if folderID <= 0 {
		return fmt.Errorf("folderId 参数非法")
	}
	_, err := s.db.ExecContext(ctx, "DELETE FROM mtphoto_folder_favorite WHERE folder_id = ?", folderID)
	return err
}

func (s *MtPhotoFolderFavoriteService) findByFolderID(ctx context.Context, folderID int64) (*MtPhotoFolderFavorite, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db not initialized")
	}
	if folderID <= 0 {
		return nil, fmt.Errorf("folderId 参数非法")
	}

	var item MtPhotoFolderFavorite
	var coverMD5, tagsJSON, note sql.NullString
	var createdAt, updatedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, folder_id, folder_name, folder_path, cover_md5, tags_json, note, created_at, updated_at
		FROM mtphoto_folder_favorite
		WHERE folder_id = ?
		LIMIT 1
	`, folderID).Scan(
		&item.ID,
		&item.FolderID,
		&item.FolderName,
		&item.FolderPath,
		&coverMD5,
		&tagsJSON,
		&note,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if coverMD5.Valid {
		item.CoverMD5 = strings.TrimSpace(coverMD5.String)
	}
	if tagsJSON.Valid {
		item.Tags = parseMtPhotoFolderFavoriteTags(tagsJSON.String)
	} else {
		item.Tags = []string{}
	}
	if note.Valid {
		item.Note = strings.TrimSpace(note.String)
	}
	item.FolderName = strings.TrimSpace(item.FolderName)
	item.FolderPath = strings.TrimSpace(item.FolderPath)
	item.CreateTime = formatNullLocalDateTimeISO(createdAt)
	item.UpdateTime = formatNullLocalDateTimeISO(updatedAt)

	return &item, nil
}
