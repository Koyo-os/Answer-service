package repository

import (
	"context"

	"github.com/Koyo-os/answer-service/internal/entity"
	"github.com/Koyo-os/answer-service/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Repository struct {
	db     *gorm.DB
	logger *logger.Logger
}

func NewRepository(db *gorm.DB, logger *logger.Logger) *Repository {
	return &Repository{
		db:     db,
		logger: logger,
	}
}

func (repo *Repository) CreateAnswer(ctx context.Context, answer *entity.Answer) error {
	res := repo.db.WithContext(ctx).Create(answer)

	if err := res.Error; err != nil {
		repo.logger.Error("error create answer",
			zap.String("answer_id", answer.ID.String()),
			zap.Error(err))

		return err
	}

	return nil
}

func (repo *Repository) DeleteAnswer(ctx context.Context, id uuid.UUID) error {
	res := repo.db.WithContext(ctx).Where("id = ?", id).Delete(&entity.Answer{})

	if err := res.Error; err != nil {
		repo.logger.Error("error delete answer",
			zap.String("answer_id", id.String()),
			zap.Error(err))

		return err
	}

	return nil
}
