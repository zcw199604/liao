package app

import (
	"context"
	"database/sql"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Identity struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Sex        string `json:"sex"`
	CreatedAt  string `json:"createdAt,omitempty"`
	LastUsedAt string `json:"lastUsedAt,omitempty"`
}

type IdentityService struct {
	db *sql.DB
}

var identityRandIntnFn = func(n int) int {
	return rand.New(rand.NewSource(time.Now().UnixNano())).Intn(n)
}

func NewIdentityService(db *sql.DB) *IdentityService {
	return &IdentityService{db: db}
}

func (s *IdentityService) GetAll(ctx context.Context) ([]Identity, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, name, sex, created_at, last_used_at FROM identity ORDER BY last_used_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Identity
	for rows.Next() {
		var id, name, sex string
		var createdAt, lastUsedAt sql.NullTime
		if err := rows.Scan(&id, &name, &sex, &createdAt, &lastUsedAt); err != nil {
			return nil, err
		}
		out = append(out, Identity{
			ID:         id,
			Name:       name,
			Sex:        sex,
			CreatedAt:  formatIdentityTime(createdAt),
			LastUsedAt: formatIdentityTime(lastUsedAt),
		})
	}
	return out, rows.Err()
}

func (s *IdentityService) GetByID(ctx context.Context, id string) (*Identity, error) {
	var name, sex string
	var createdAt, lastUsedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, "SELECT name, sex, created_at, last_used_at FROM identity WHERE id = ?", id).Scan(&name, &sex, &createdAt, &lastUsedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &Identity{
		ID:         id,
		Name:       name,
		Sex:        sex,
		CreatedAt:  formatIdentityTime(createdAt),
		LastUsedAt: formatIdentityTime(lastUsedAt),
	}, nil
}

func (s *IdentityService) Create(ctx context.Context, name string, sex string) (*Identity, error) {
	id := strings.ReplaceAll(uuid.NewString(), "-", "")
	now := time.Now().Format("2006-01-02 15:04:05")

	if _, err := s.db.ExecContext(ctx, "INSERT INTO identity (id, name, sex, created_at, last_used_at) VALUES (?, ?, ?, ?, ?)", id, name, sex, now, now); err != nil {
		return nil, err
	}

	return &Identity{
		ID:         id,
		Name:       name,
		Sex:        sex,
		CreatedAt:  now,
		LastUsedAt: now,
	}, nil
}

func (s *IdentityService) QuickCreate(ctx context.Context) (*Identity, error) {
	name := randomIdentityName()
	sex := "女"
	if identityRandIntnFn(2) == 0 {
		sex = "男"
	}
	return s.Create(ctx, name, sex)
}

func (s *IdentityService) Update(ctx context.Context, id, name, sex string) (*Identity, error) {
	existing, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	if _, err := s.db.ExecContext(ctx, "UPDATE identity SET name = ?, sex = ?, last_used_at = ? WHERE id = ?", name, sex, now, id); err != nil {
		return nil, err
	}

	existing.Name = name
	existing.Sex = sex
	existing.LastUsedAt = now
	return existing, nil
}

func (s *IdentityService) UpdateLastUsedAt(ctx context.Context, id string) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := s.db.ExecContext(ctx, "UPDATE identity SET last_used_at = ? WHERE id = ?", now, id)
	return err
}

func (s *IdentityService) Delete(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, "DELETE FROM identity WHERE id = ?", id)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

func (s *IdentityService) UpdateID(ctx context.Context, oldID, newID, name, sex string) (*Identity, error) {
	oldIdentity, err := s.GetByID(ctx, oldID)
	if err != nil {
		return nil, err
	}
	if oldIdentity == nil {
		return nil, nil
	}

	existingNew, err := s.GetByID(ctx, newID)
	if err != nil {
		return nil, err
	}
	if existingNew != nil {
		return nil, nil
	}

	createdAt := oldIdentity.CreatedAt
	now := time.Now().Format("2006-01-02 15:04:05")

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, "DELETE FROM identity WHERE id = ?", oldID); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO identity (id, name, sex, created_at, last_used_at) VALUES (?, ?, ?, ?, ?)", newID, name, sex, createdAt, now); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &Identity{
		ID:         newID,
		Name:       name,
		Sex:        sex,
		CreatedAt:  createdAt,
		LastUsedAt: now,
	}, nil
}

func formatIdentityTime(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format("2006-01-02 15:04:05")
}

var identityRandomNames = []string{
	"小明", "小红", "小华", "小美", "小强", "小芳", "小军", "小丽",
	"阿杰", "阿伟", "阿珍", "阿莲", "阿辉", "阿英", "阿龙", "阿凤",
	"大熊", "小鹿", "白兔", "黑猫", "金鱼", "银狐", "青蛙", "蝴蝶",
	"星辰", "月光", "阳光", "彩虹", "流云", "清风", "细雨", "白雪",
	"晨曦", "暮色", "春风", "夏雨", "秋叶", "冬雪", "花开", "叶落",
	"海浪", "山峰", "森林", "草原", "沙漠", "冰川", "火焰", "闪电",
	"孤独", "寂寞", "快乐", "忧伤", "温柔", "坚强", "勇敢", "善良",
	"路人甲", "过客乙", "行者丙", "旅人丁", "浪子", "游侠", "隐士", "剑客",
}

func randomIdentityName() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return identityRandomNames[r.Intn(len(identityRandomNames))]
}
