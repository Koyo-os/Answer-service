package repository

import (
	"context"
	"fmt"

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

func (repo *Repository) GetAnswerBy(ctx context.Context, key string, value interface{}) ([]entity.Answer, error) {
	var answers []entity.Answer

	res := repo.db.WithContext(ctx).Where(fmt.Sprintf("%s = ?", key), value).Find(&answers)

	if err := res.Error; err != nil {
		repo.logger.Error("error get answer",
			zap.String("key", key),
			zap.Error(err))

		return nil, err
	}

	return answers, nil
}

func (repo *Repository) GetAnswer(ctx context.Context, id uuid.UUID) (*entity.Answer, error) {
	answer := new(entity.Answer)

	res := repo.db.WithContext(ctx).Find(answer)

	if err := res.Error; err != nil {
		repo.logger.Error("error get answer",
			zap.String("answer_id", id.String()),
			zap.Error(err))

		return nil, err
	}

	return answer, nil
}
