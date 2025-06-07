package service

import (
	"context"

	"github.com/Koyo-os/answer-service/internal/entity"
	"github.com/google/uuid"
)

type (
	Repository interface {
		CreateAnswer(context.Context, *entity.Answer) error
		DeleteAnswer(context.Context, uuid.UUID) error
	}

	Publisher interface {
		Publish(any, string) error
	}

	Casher interface {
		DoCashing(context.Context, string, any) error // payload must to be pointer
		DeleteFromCash(context.Context, string) error
	}
)
