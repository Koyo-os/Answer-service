package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Koyo-os/answer-service/internal/entity"
	"github.com/Koyo-os/answer-service/pkg/retrier"
	"github.com/google/uuid"
)

const (
	DefaultRetrierAttempts = 3
	DefaultRetryDelay      = 5 * time.Second
	AnswerKeyTemplate      = "answer:%s"
)

const (
	AnswerCreatedEventType = "answer.created"
	AnswerDeletedEventType = "answer.deleted"
)

var (
	ErrAnswerNil = errors.New("answer cannot be nil")
	ErrInvalidID = errors.New("invalid answer ID format")
)

type Service struct {
	casher     Casher
	publisher  Publisher
	repository Repository
	timeout    time.Duration
}

type DeletePayload struct {
	ID string `json:"id"`
}

func NewService(casher Casher, publisher Publisher, repo Repository, timeout time.Duration) *Service {
	return &Service{
		casher:     casher,
		publisher:  publisher,
		repository: repo,
		timeout:    timeout,
	}
}

func (s *Service) getContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), s.timeout)
}

func (s *Service) Add(answer *entity.Answer) error {
	if answer == nil {
		return ErrAnswerNil
	}

	ctx, cancel := s.getContext()
	defer cancel()

	if err := s.repository.CreateAnswer(ctx, answer); err != nil {
		return fmt.Errorf("failed to create answer: %w", err)
	}

	// Execute cache and publish operations concurrently
	if err := s.executeAsyncOperations(
		s.createCacheOperation(answer),
		s.createPublishOperation(answer, AnswerCreatedEventType),
	); err != nil {
		return fmt.Errorf("failed to complete async operations: %w", err)
	}

	return nil
}

func (s *Service) Delete(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidID, id)
	}

	ctx, cancel := s.getContext()
	defer cancel()

	if err := s.repository.DeleteAnswer(ctx, uid); err != nil {
		return fmt.Errorf("failed to delete answer: %w", err)
	}

	deletePayload := &DeletePayload{ID: id}

	// Execute cache deletion and publish operations concurrently
	if err := s.executeAsyncOperations(
		s.createCacheDeleteOperation(id),
		s.createPublishOperation(deletePayload, AnswerDeletedEventType),
	); err != nil {
		return fmt.Errorf("failed to complete async operations: %w", err)
	}

	return nil
}

// executeAsyncOperations runs multiple operations concurrently and returns the first error encountered
func (s *Service) executeAsyncOperations(operations ...func() error) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(operations))

	for _, operation := range operations {
		wg.Add(1)
		go func(op func() error) {
			defer wg.Done()
			if err := op(); err != nil {
				errChan <- err
			}
		}(operation)
	}

	// Wait for all operations to complete
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Return the first error encountered
	for err := range errChan {
		return err
	}

	return nil
}

// createCacheOperation creates a cache operation with retry logic
func (s *Service) createCacheOperation(answer *entity.Answer) func() error {
	return func() error {
		ctx, cancel := s.getContext()
		defer cancel()

		return retrier.Do(DefaultRetrierAttempts, DefaultRetryDelay, func() error {
			key := fmt.Sprintf(AnswerKeyTemplate, answer.ID.String())
			return s.casher.DoCashing(ctx, key, answer)
		})
	}
}

// createCacheDeleteOperation creates a cache deletion operation with retry logic
func (s *Service) createCacheDeleteOperation(id string) func() error {
	return func() error {
		ctx, cancel := s.getContext()
		defer cancel()

		return retrier.Do(DefaultRetrierAttempts, DefaultRetryDelay, func() error {
			key := fmt.Sprintf(AnswerKeyTemplate, id)
			return s.casher.DeleteFromCash(ctx, key)
		})
	}
}

// createPublishOperation creates a publish operation with retry logic
func (s *Service) createPublishOperation(payload interface{}, eventType string) func() error {
	return func() error {
		return retrier.Do(DefaultRetrierAttempts, DefaultRetryDelay, func() error {
			return s.publisher.Publish(payload, eventType)
		})
	}
}
