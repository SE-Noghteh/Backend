package postgres

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RevokedAccessTokenModel struct {
	TokenID   string    `gorm:"primaryKey;size:32"`
	ExpiresAt time.Time `gorm:"not null;index"`
}

func (RevokedAccessTokenModel) TableName() string { return "revoked_access_tokens" }

type TokenRevocationRepository struct{ db *gorm.DB }

func NewTokenRevocationRepository(db *gorm.DB) *TokenRevocationRepository {
	return &TokenRevocationRepository{db: db}
}

func (r *TokenRevocationRepository) IsRevoked(ctx context.Context, tokenID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&RevokedAccessTokenModel{}).Where("token_id = ?", tokenID).Count(&count).Error
	return count > 0, err
}

func (r *TokenRevocationRepository) Revoke(ctx context.Context, tokenID string, expiresAt time.Time) error {
	model := RevokedAccessTokenModel{TokenID: tokenID, ExpiresAt: expiresAt}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&model).Error
}
