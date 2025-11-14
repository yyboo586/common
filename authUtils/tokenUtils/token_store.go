package tokenUtils

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
)

// TokenStoreConfig 数据库存储配置
type TokenStoreConfig struct {
	DSN         string
	Group       string
	TableName   string
	EnableDebug bool
}

// DefaultTokenStoreConfig 默认配置
func DefaultTokenStoreConfig() *TokenStoreConfig {
	return &TokenStoreConfig{
		Group:     "default",
		TableName: "t_token",
	}
}

// TokenStore 负责持久化每个颁发的 JWT
type TokenStore struct {
	db        gdb.DB
	tableName string
}

// NewTokenStore 创建 TokenStore
func NewTokenStore(cfg *TokenStoreConfig) (*TokenStore, error) {
	if cfg == nil {
		cfg = DefaultTokenStoreConfig()
	}
	if cfg.Group == "" {
		cfg.Group = "default"
	}
	if cfg.TableName == "" {
		cfg.TableName = "t_token"
	}

	gdb.SetConfig(gdb.Config{
		cfg.Group: gdb.ConfigGroup{
			{Link: cfg.DSN},
		},
	})

	db := g.DB(cfg.Group)
	if db == nil {
		return nil, gerror.Newf("failed to get database instance for group: %s", cfg.Group)
	}
	if cfg.EnableDebug {
		db.SetDebug(true)
	}

	return &TokenStore{
		db:        db,
		tableName: cfg.TableName,
	}, nil
}

// EnsureTable 确保存储表存在
func (s *TokenStore) EnsureTable(ctx context.Context) error {
	createTableSQL := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
    id VARCHAR(40) NOT NULL COMMENT '令牌唯一标识JTI',
    user_id VARCHAR(40) NOT NULL COMMENT '用户ID',
    device_id VARCHAR(40) NOT NULL DEFAULT '' COMMENT '设备ID',
    content TEXT COMMENT '自定义数据',
    token_type VARCHAR(20) NOT NULL DEFAULT 'access' COMMENT '令牌类型(access:访问令牌,refresh:刷新令牌)',
    refresh_token_id VARCHAR(40) NOT NULL DEFAULT '' COMMENT '关联的刷新令牌ID(访问令牌)或访问令牌ID(刷新令牌)',
    is_active TINYINT(4) NOT NULL DEFAULT 1 COMMENT '令牌有效标识(0:无效,1:有效)',
	expire_time BIGINT(20) NOT NULL COMMENT '令牌过期时间',
    create_time BIGINT(20) NOT NULL COMMENT '令牌创建时间',
	update_time BIGINT(20) NOT NULL COMMENT '令牌更新时间',
    PRIMARY KEY (id),
    KEY idx_user_id (user_id),
    KEY idx_device_id (device_id),
    KEY idx_refresh_token_id (refresh_token_id)
) ENGINE=InnoDB COMMENT='令牌表';`, s.tableName)

	_, err := s.db.Exec(ctx, createTableSQL)
	return err
}

// Create 写入令牌记录
func (s *TokenStore) Create(ctx context.Context, jti, userID, deviceID, content string, expireTime int64) error {
	return s.CreateWithType(ctx, jti, userID, deviceID, content, "access", "", expireTime)
}

// CreateWithType 写入令牌记录（支持指定类型和关联令牌）
func (s *TokenStore) CreateWithType(ctx context.Context, jti, userID, deviceID, content, tokenType, refreshTokenID string, expireTime int64) error {
	data := g.Map{
		"id":               jti,
		"user_id":          userID,
		"device_id":        deviceID,
		"content":          content,
		"token_type":       tokenType,
		"refresh_token_id": refreshTokenID,
		"is_active":        1,
		"expire_time":      expireTime,
		"create_time":      time.Now().Unix(),
		"update_time":      time.Now().Unix(),
	}

	_, err := s.db.Model(s.tableName).Ctx(ctx).Data(data).Insert()
	return err
}

// GetActive 根据 JTI 查询有效令牌
func (s *TokenStore) GetActive(ctx context.Context, jti string) (*TokenEntity, error) {
	var record TokenEntity
	err := s.db.Model(s.tableName).
		Ctx(ctx).
		Where("id = ? AND is_active = ?", jti, 1).
		Scan(&record)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &record, nil
}

// GetActiveRefreshToken 根据刷新令牌JTI查询有效的刷新令牌
func (s *TokenStore) GetActiveRefreshToken(ctx context.Context, refreshTokenJTI string) (*TokenEntity, error) {
	var record TokenEntity
	err := s.db.Model(s.tableName).
		Ctx(ctx).
		Where("id = ? AND token_type = ? AND is_active = ?", refreshTokenJTI, "refresh", 1).
		Scan(&record)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &record, nil
}

// GetAccessTokenByRefreshTokenID 根据刷新令牌ID查询关联的访问令牌
func (s *TokenStore) GetAccessTokenByRefreshTokenID(ctx context.Context, refreshTokenID string) (*TokenEntity, error) {
	var record TokenEntity
	err := s.db.Model(s.tableName).
		Ctx(ctx).
		Where("refresh_token_id = ? AND token_type = ?", refreshTokenID, "access").
		Scan(&record)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &record, nil
}

// DeactivateByJTI 令牌主动失效
func (s *TokenStore) RevokeByJTI(ctx context.Context, jti string) error {
	return s.deactivate(ctx, g.Map{"id": jti})
}

// DeactivateByDevice 禁用设备下所有令牌
func (s *TokenStore) RevokeByDeviceID(ctx context.Context, deviceID string) error {
	return s.deactivate(ctx, g.Map{"device_id": deviceID})
}

// DeactivateByUser 禁用用户下所有令牌
func (s *TokenStore) DeactivateByUser(ctx context.Context, userID string) error {
	return s.deactivate(ctx, g.Map{"user_id": userID})
}

func (s *TokenStore) deactivate(ctx context.Context, where g.Map) error {
	if len(where) == 0 {
		return nil
	}

	_, err := s.db.Model(s.tableName).Ctx(ctx).
		Data(g.Map{
			"is_active":   0,
			"update_time": time.Now().Unix(),
		}).
		Where(where).
		Update()
	return err
}
