package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"liao/internal/database"
)

// UserArchiveListSource 标识用户归档来源列表类型。
type UserArchiveListSource string

const (
	UserArchiveListSourceHistory  UserArchiveListSource = "history"
	UserArchiveListSourceFavorite UserArchiveListSource = "favorite"
)

// UserArchiveService 定义用户归档能力（持久化+合并回填）。
type UserArchiveService interface {
	PersistUserList(ctx context.Context, ownerUserID string, users []map[string]any, source UserArchiveListSource)
	MergeArchivedUsers(ctx context.Context, ownerUserID string, upstream []map[string]any, source UserArchiveListSource) []map[string]any
	TouchConversation(ctx context.Context, ownerUserID, targetUserID string)
	SaveLastMessage(ctx context.Context, ownerUserID, targetUserID, content, messageTime string)
	DeleteConversation(ctx context.Context, ownerUserID, targetUserID string)
}

// DBUserArchiveService 基于数据库实现 UserArchiveService。
type DBUserArchiveService struct {
	db *database.DB
}

// NewDBUserArchiveService 创建数据库归档服务。
func NewDBUserArchiveService(db *database.DB) *DBUserArchiveService {
	if db == nil {
		return nil
	}
	return &DBUserArchiveService{db: db}
}

func (s *DBUserArchiveService) PersistUserList(ctx context.Context, ownerUserID string, users []map[string]any, source UserArchiveListSource) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	if s == nil || s.db == nil || ownerUserID == "" || len(users) == 0 {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	historyFlag, favoriteFlag := sourceFlags(source)
	now := time.Now()
	prepared := make([]archiveUpsertInput, 0, len(users))
	for _, user := range users {
		targetUserID := strings.TrimSpace(extractUserID(user))
		if targetUserID == "" {
			continue
		}

		snapshot := sanitizeArchivedUserSnapshot(user, targetUserID)
		snapshotRaw, err := json.Marshal(snapshot)
		if err != nil {
			slog.Warn("归档用户快照序列化失败", "ownerUserID", ownerUserID, "targetUserID", targetUserID, "error", err)
			continue
		}

		lastMsg := strings.TrimSpace(toString(snapshot["lastMsg"]))
		lastTime := strings.TrimSpace(toString(snapshot["lastTime"]))

		prepared = append(prepared, archiveUpsertInput{
			OwnerUserID:    ownerUserID,
			TargetUserID:   targetUserID,
			SnapshotJSON:   string(snapshotRaw),
			LastMsg:        lastMsg,
			LastTime:       lastTime,
			SeenInHistory:  historyFlag,
			SeenInFavorite: favoriteFlag,
			SeenAt:         now,
		})
	}
	if len(prepared) == 0 {
		return
	}

	start := time.Now()
	written, skipped, err := s.upsertRowsForList(ctx, prepared)
	durationMs := time.Since(start).Milliseconds()
	if err != nil {
		slog.Warn("归档用户列表失败", "ownerUserID", ownerUserID, "source", source, "preparedSize", len(prepared), "writtenSize", written, "skippedSize", skipped, "durationMs", durationMs, "error", err)
		return
	}
	slog.Info("[timing] archive.persistUserList", "ownerUserID", ownerUserID, "source", source, "preparedSize", len(prepared), "writtenSize", written, "skippedSize", skipped, "durationMs", durationMs)
}

func (s *DBUserArchiveService) MergeArchivedUsers(ctx context.Context, ownerUserID string, upstream []map[string]any, source UserArchiveListSource) []map[string]any {
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return upstream
	}
	if s == nil || s.db == nil {
		return upstream
	}
	if ctx == nil {
		ctx = context.Background()
	}

	rows, err := s.listArchivedRows(ctx, ownerUserID, source)
	if err != nil {
		slog.Warn("读取归档用户失败", "ownerUserID", ownerUserID, "source", source, "error", err)
		return upstream
	}
	if len(rows) == 0 {
		return upstream
	}

	out := make([]map[string]any, 0, len(upstream)+len(rows))
	seen := make(map[string]struct{}, len(upstream)+len(rows))

	for _, item := range upstream {
		if item == nil {
			continue
		}
		targetUserID := strings.TrimSpace(extractUserID(item))
		if targetUserID != "" {
			seen[targetUserID] = struct{}{}
		}
		out = append(out, item)
	}

	for _, row := range rows {
		targetUserID := strings.TrimSpace(row.TargetUserID)
		if targetUserID == "" {
			continue
		}
		if _, ok := seen[targetUserID]; ok {
			continue
		}

		merged := map[string]any{}
		if strings.TrimSpace(row.SnapshotJSON) != "" {
			if err := json.Unmarshal([]byte(row.SnapshotJSON), &merged); err != nil {
				slog.Warn("反序列化归档用户快照失败", "ownerUserID", ownerUserID, "targetUserID", targetUserID, "error", err)
				merged = map[string]any{}
			}
		}
		if merged == nil {
			merged = map[string]any{}
		}

		ensureArchivedUserID(merged, targetUserID)
		if strings.TrimSpace(toString(merged["lastMsg"])) == "" && strings.TrimSpace(row.LastMsg) != "" {
			merged["lastMsg"] = row.LastMsg
		}
		if strings.TrimSpace(toString(merged["lastTime"])) == "" && strings.TrimSpace(row.LastTime) != "" {
			merged["lastTime"] = row.LastTime
		}
		merged["localArchived"] = true

		out = append(out, merged)
		seen[targetUserID] = struct{}{}
	}

	return out
}

func (s *DBUserArchiveService) TouchConversation(ctx context.Context, ownerUserID, targetUserID string) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	targetUserID = strings.TrimSpace(targetUserID)
	if s == nil || s.db == nil || ownerUserID == "" || targetUserID == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	now := time.Now()
	if err := s.upsertRow(ctx, archiveUpsertInput{
		OwnerUserID:   ownerUserID,
		TargetUserID:  targetUserID,
		SeenInHistory: 1,
		SeenAt:        now,
	}); err != nil {
		slog.Warn("归档会话触达失败", "ownerUserID", ownerUserID, "targetUserID", targetUserID, "error", err)
	}
}

func (s *DBUserArchiveService) SaveLastMessage(ctx context.Context, ownerUserID, targetUserID, content, messageTime string) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	targetUserID = strings.TrimSpace(targetUserID)
	content = strings.TrimSpace(content)
	messageTime = strings.TrimSpace(messageTime)
	if s == nil || s.db == nil || ownerUserID == "" || targetUserID == "" {
		return
	}
	if content == "" && messageTime == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	now := time.Now()
	if err := s.upsertRow(ctx, archiveUpsertInput{
		OwnerUserID:   ownerUserID,
		TargetUserID:  targetUserID,
		LastMsg:       content,
		LastTime:      messageTime,
		SeenInHistory: 1,
		SeenAt:        now,
	}); err != nil {
		slog.Warn("归档最后消息失败", "ownerUserID", ownerUserID, "targetUserID", targetUserID, "error", err)
	}
}

func (s *DBUserArchiveService) DeleteConversation(ctx context.Context, ownerUserID, targetUserID string) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	targetUserID = strings.TrimSpace(targetUserID)
	if s == nil || s.db == nil || ownerUserID == "" || targetUserID == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if _, err := s.db.ExecContext(
		ctx,
		"DELETE FROM chat_user_archive WHERE owner_user_id = ? AND target_user_id = ?",
		ownerUserID,
		targetUserID,
	); err != nil {
		slog.Warn("删除归档会话失败", "ownerUserID", ownerUserID, "targetUserID", targetUserID, "error", err)
	}
}

type archiveUpsertInput struct {
	OwnerUserID    string
	TargetUserID   string
	SnapshotJSON   string
	LastMsg        string
	LastTime       string
	SeenInHistory  int
	SeenInFavorite int
	SeenAt         time.Time
}

type archiveRow struct {
	TargetUserID string
	SnapshotJSON string
	LastMsg      string
	LastTime     string
}

type archiveExistingState struct {
	TargetUserID   string
	SeenInHistory  int
	SeenInFavorite int
	SnapshotJSON   string
	LastMsg        string
	LastTime       string
}

func sourceFlags(source UserArchiveListSource) (historyFlag int, favoriteFlag int) {
	switch source {
	case UserArchiveListSourceFavorite:
		return 0, 1
	default:
		return 1, 0
	}
}

func sourceFilterColumn(source UserArchiveListSource) string {
	switch source {
	case UserArchiveListSourceFavorite:
		return "seen_in_favorite"
	default:
		return "seen_in_history"
	}
}

func sanitizeArchivedUserSnapshot(user map[string]any, targetUserID string) map[string]any {
	if user == nil {
		return map[string]any{"id": targetUserID}
	}

	clean := make(map[string]any, len(user)+1)
	for k, v := range user {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		if key == "localArchived" {
			continue
		}
		clean[key] = v
	}
	ensureArchivedUserID(clean, targetUserID)
	return clean
}

func ensureArchivedUserID(user map[string]any, targetUserID string) {
	if user == nil {
		return
	}
	if strings.TrimSpace(toString(user["id"])) != "" {
		return
	}
	if strings.TrimSpace(toString(user["UserID"])) != "" {
		user["id"] = strings.TrimSpace(toString(user["UserID"]))
		return
	}
	if strings.TrimSpace(toString(user["userid"])) != "" {
		user["id"] = strings.TrimSpace(toString(user["userid"]))
		return
	}
	if strings.TrimSpace(toString(user["userId"])) != "" {
		user["id"] = strings.TrimSpace(toString(user["userId"]))
		return
	}
	if strings.TrimSpace(targetUserID) != "" {
		user["id"] = strings.TrimSpace(targetUserID)
	}
}

func nullableString(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}

func mergeArchiveUpsertInput(base, incoming archiveUpsertInput) archiveUpsertInput {
	if strings.TrimSpace(base.TargetUserID) == "" {
		return incoming
	}
	if strings.TrimSpace(incoming.SnapshotJSON) != "" {
		base.SnapshotJSON = incoming.SnapshotJSON
	}
	if strings.TrimSpace(incoming.LastMsg) != "" {
		base.LastMsg = incoming.LastMsg
	}
	if strings.TrimSpace(incoming.LastTime) != "" {
		base.LastTime = incoming.LastTime
	}
	if incoming.SeenInHistory == 1 {
		base.SeenInHistory = 1
	}
	if incoming.SeenInFavorite == 1 {
		base.SeenInFavorite = 1
	}
	if incoming.SeenAt.After(base.SeenAt) {
		base.SeenAt = incoming.SeenAt
	}
	return base
}

func mergeSeenFlag(existing int, incoming int) int {
	if existing == 1 || incoming == 1 {
		return 1
	}
	return 0
}

func archiveFieldChanged(incoming, existing string) bool {
	incoming = strings.TrimSpace(incoming)
	existing = strings.TrimSpace(existing)
	if incoming == "" {
		return false
	}
	return incoming != existing
}

func archiveRowNeedsUpdate(in archiveUpsertInput, existing archiveExistingState) bool {
	if in.SeenInHistory != existing.SeenInHistory {
		return true
	}
	if in.SeenInFavorite != existing.SeenInFavorite {
		return true
	}
	if archiveFieldChanged(in.SnapshotJSON, existing.SnapshotJSON) {
		return true
	}
	if archiveFieldChanged(in.LastMsg, existing.LastMsg) {
		return true
	}
	if archiveFieldChanged(in.LastTime, existing.LastTime) {
		return true
	}
	return false
}

func (s *DBUserArchiveService) upsertRowsForList(ctx context.Context, rows []archiveUpsertInput) (written int, skipped int, err error) {
	if s == nil || s.db == nil || len(rows) == 0 {
		return 0, 0, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	deduped := make(map[string]archiveUpsertInput, len(rows))
	targetIDs := make([]string, 0, len(rows))
	ownerUserID := ""
	for _, row := range rows {
		row.OwnerUserID = strings.TrimSpace(row.OwnerUserID)
		row.TargetUserID = strings.TrimSpace(row.TargetUserID)
		if row.OwnerUserID == "" || row.TargetUserID == "" {
			skipped++
			continue
		}
		if ownerUserID == "" {
			ownerUserID = row.OwnerUserID
		}
		if ownerUserID != row.OwnerUserID {
			return 0, skipped, fmt.Errorf("owner_user_id mismatch in batch upsert")
		}
		if row.SeenAt.IsZero() {
			row.SeenAt = time.Now()
		}
		if existing, ok := deduped[row.TargetUserID]; ok {
			deduped[row.TargetUserID] = mergeArchiveUpsertInput(existing, row)
			continue
		}
		deduped[row.TargetUserID] = row
		targetIDs = append(targetIDs, row.TargetUserID)
	}
	if ownerUserID == "" || len(targetIDs) == 0 {
		return 0, skipped, nil
	}

	existingStates, err := s.fetchExistingStates(ctx, ownerUserID, targetIDs)
	if err != nil {
		return 0, skipped, err
	}

	toWrite := make([]archiveUpsertInput, 0, len(targetIDs))
	for _, targetID := range targetIDs {
		row := deduped[targetID]
		existing, ok := existingStates[targetID]
		if ok {
			row.SeenInHistory = mergeSeenFlag(existing.SeenInHistory, row.SeenInHistory)
			row.SeenInFavorite = mergeSeenFlag(existing.SeenInFavorite, row.SeenInFavorite)
			if !archiveRowNeedsUpdate(row, existing) {
				skipped++
				continue
			}
		}
		toWrite = append(toWrite, row)
	}
	if len(toWrite) == 0 {
		return 0, skipped, nil
	}
	if err := s.upsertRowsBatch(ctx, toWrite); err != nil {
		return 0, skipped, err
	}
	return len(toWrite), skipped, nil
}

func (s *DBUserArchiveService) fetchExistingStates(ctx context.Context, ownerUserID string, targetUserIDs []string) (map[string]archiveExistingState, error) {
	result := make(map[string]archiveExistingState, len(targetUserIDs))
	if s == nil || s.db == nil || strings.TrimSpace(ownerUserID) == "" || len(targetUserIDs) == 0 {
		return result, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	query, args, err := database.ExpandIn(
		`SELECT target_user_id, seen_in_history, seen_in_favorite, COALESCE(snapshot_json, ''), COALESCE(last_msg, ''), COALESCE(last_time, '')
		FROM chat_user_archive
		WHERE owner_user_id = ? AND target_user_id IN (?)`,
		ownerUserID,
		targetUserIDs,
	)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var state archiveExistingState
		if err := rows.Scan(&state.TargetUserID, &state.SeenInHistory, &state.SeenInFavorite, &state.SnapshotJSON, &state.LastMsg, &state.LastTime); err != nil {
			return nil, err
		}
		state.TargetUserID = strings.TrimSpace(state.TargetUserID)
		result[state.TargetUserID] = state
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *DBUserArchiveService) upsertRowsBatch(ctx context.Context, rows []archiveUpsertInput) error {
	if s == nil || s.db == nil || len(rows) == 0 {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	values := make([]string, 0, len(rows))
	args := make([]any, 0, len(rows)*11)
	for _, row := range rows {
		now := row.SeenAt
		if now.IsZero() {
			now = time.Now()
		}
		values = append(values, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		args = append(
			args,
			row.OwnerUserID,
			row.TargetUserID,
			nullableString(row.SnapshotJSON),
			nullableString(row.LastMsg),
			nullableString(row.LastTime),
			row.SeenInHistory,
			row.SeenInFavorite,
			now,
			now,
			now,
			now,
		)
	}

	baseQuery := `INSERT INTO chat_user_archive (
		owner_user_id, target_user_id, snapshot_json, last_msg, last_time,
		seen_in_history, seen_in_favorite, first_seen_at, last_seen_at, created_at, updated_at
	) VALUES ` + strings.Join(values, ",")

	var query string
	switch s.db.Dialect().Name() {
	case "postgres":
		query = baseQuery + `
		ON CONFLICT (owner_user_id, target_user_id) DO UPDATE SET
			snapshot_json = COALESCE(EXCLUDED.snapshot_json, chat_user_archive.snapshot_json),
			last_msg = COALESCE(EXCLUDED.last_msg, chat_user_archive.last_msg),
			last_time = COALESCE(EXCLUDED.last_time, chat_user_archive.last_time),
			seen_in_history = GREATEST(chat_user_archive.seen_in_history, EXCLUDED.seen_in_history),
			seen_in_favorite = GREATEST(chat_user_archive.seen_in_favorite, EXCLUDED.seen_in_favorite),
			first_seen_at = LEAST(chat_user_archive.first_seen_at, EXCLUDED.first_seen_at),
			last_seen_at = GREATEST(chat_user_archive.last_seen_at, EXCLUDED.last_seen_at),
			updated_at = EXCLUDED.updated_at`
	default:
		query = baseQuery + `
		ON DUPLICATE KEY UPDATE
			snapshot_json = COALESCE(VALUES(snapshot_json), snapshot_json),
			last_msg = COALESCE(VALUES(last_msg), last_msg),
			last_time = COALESCE(VALUES(last_time), last_time),
			seen_in_history = GREATEST(seen_in_history, VALUES(seen_in_history)),
			seen_in_favorite = GREATEST(seen_in_favorite, VALUES(seen_in_favorite)),
			first_seen_at = LEAST(first_seen_at, VALUES(first_seen_at)),
			last_seen_at = GREATEST(last_seen_at, VALUES(last_seen_at)),
			updated_at = VALUES(updated_at)`
	}

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *DBUserArchiveService) upsertRow(ctx context.Context, in archiveUpsertInput) error {
	if ctx == nil {
		ctx = context.Background()
	}

	ownerUserID := strings.TrimSpace(in.OwnerUserID)
	targetUserID := strings.TrimSpace(in.TargetUserID)
	if ownerUserID == "" || targetUserID == "" {
		return nil
	}

	now := in.SeenAt
	if now.IsZero() {
		now = time.Now()
	}

	existingHistory, existingFavorite, exists, err := s.fetchSeenFlags(ctx, ownerUserID, targetUserID)
	if err != nil {
		return err
	}

	historyFlag := existingHistory
	if !exists {
		historyFlag = 0
	}
	if in.SeenInHistory == 1 {
		historyFlag = 1
	}

	favoriteFlag := existingFavorite
	if !exists {
		favoriteFlag = 0
	}
	if in.SeenInFavorite == 1 {
		favoriteFlag = 1
	}

	if !exists {
		_, err := s.db.ExecContext(
			ctx,
			`INSERT INTO chat_user_archive (
				owner_user_id, target_user_id, snapshot_json, last_msg, last_time,
				seen_in_history, seen_in_favorite, first_seen_at, last_seen_at, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			ownerUserID,
			targetUserID,
			nullableString(in.SnapshotJSON),
			nullableString(in.LastMsg),
			nullableString(in.LastTime),
			historyFlag,
			favoriteFlag,
			now,
			now,
			now,
			now,
		)
		if err != nil {
			if !s.db.Dialect().IsDuplicateKey(err) {
				return err
			}
			// 并发写入下重试 UPDATE。
			_, err = s.db.ExecContext(
				ctx,
				`UPDATE chat_user_archive
				SET
					snapshot_json = COALESCE(?, snapshot_json),
					last_msg = COALESCE(?, last_msg),
					last_time = COALESCE(?, last_time),
					seen_in_history = ?,
					seen_in_favorite = ?,
					last_seen_at = ?,
					updated_at = ?
				WHERE owner_user_id = ? AND target_user_id = ?`,
				nullableString(in.SnapshotJSON),
				nullableString(in.LastMsg),
				nullableString(in.LastTime),
				historyFlag,
				favoriteFlag,
				now,
				now,
				ownerUserID,
				targetUserID,
			)
		}
		return err
	}

	_, err = s.db.ExecContext(
		ctx,
		`UPDATE chat_user_archive
		SET
			snapshot_json = COALESCE(?, snapshot_json),
			last_msg = COALESCE(?, last_msg),
			last_time = COALESCE(?, last_time),
			seen_in_history = ?,
			seen_in_favorite = ?,
			last_seen_at = ?,
			updated_at = ?
		WHERE owner_user_id = ? AND target_user_id = ?`,
		nullableString(in.SnapshotJSON),
		nullableString(in.LastMsg),
		nullableString(in.LastTime),
		historyFlag,
		favoriteFlag,
		now,
		now,
		ownerUserID,
		targetUserID,
	)
	return err
}

func (s *DBUserArchiveService) fetchSeenFlags(ctx context.Context, ownerUserID, targetUserID string) (history int, favorite int, exists bool, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	row := s.db.QueryRowContext(
		ctx,
		"SELECT seen_in_history, seen_in_favorite FROM chat_user_archive WHERE owner_user_id = ? AND target_user_id = ? LIMIT 1",
		ownerUserID,
		targetUserID,
	)
	if scanErr := row.Scan(&history, &favorite); scanErr != nil {
		if scanErr == sql.ErrNoRows {
			return 0, 0, false, nil
		}
		return 0, 0, false, scanErr
	}
	return history, favorite, true, nil
}

func (s *DBUserArchiveService) listArchivedRows(ctx context.Context, ownerUserID string, source UserArchiveListSource) ([]archiveRow, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	query := `
		SELECT target_user_id, snapshot_json, last_msg, last_time
		FROM chat_user_archive
		WHERE owner_user_id = ? AND ` + sourceFilterColumn(source) + ` = 1
		ORDER BY updated_at DESC, id DESC
	`
	rows, err := s.db.QueryContext(ctx, query, ownerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]archiveRow, 0)
	for rows.Next() {
		var (
			targetUserID string
			snapshotRaw  sql.NullString
			lastMsg      sql.NullString
			lastTime     sql.NullString
		)
		if err := rows.Scan(&targetUserID, &snapshotRaw, &lastMsg, &lastTime); err != nil {
			return nil, err
		}
		result = append(result, archiveRow{
			TargetUserID: strings.TrimSpace(targetUserID),
			SnapshotJSON: strings.TrimSpace(snapshotRaw.String),
			LastMsg:      strings.TrimSpace(lastMsg.String),
			LastTime:     strings.TrimSpace(lastTime.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
