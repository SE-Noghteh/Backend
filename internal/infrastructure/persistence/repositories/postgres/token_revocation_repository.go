package postgres

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"zedlancer/backend/internal/infrastructure/persistence/models"
)

type TokenRevocationRepository struct{ db *gorm.DB }

func NewTokenRevocationRepository(db *gorm.DB) *TokenRevocationRepository {
	return &TokenRevocationRepository{db: db}
}

func (r *TokenRevocationRepository) IsRevoked(ctx context.Context, tokenID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.RevokedAccessTokenModel{}).Where("token_id = ?", tokenID).Count(&count).Error
	return count > 0, err
}

func (r *TokenRevocationRepository) Revoke(ctx context.Context, tokenID string, expiresAt time.Time) error {
	model := models.RevokedAccessTokenModel{TokenID: tokenID, ExpiresAt: expiresAt}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&model).Error
}
