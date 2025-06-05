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
	DefaultRetryDelay = 5
	AnswerKeyTemplate = "answer:%s"
)

type Service struct{
	casher Casher
	publisher Publisher
	repository Repository

	timeout time.Duration
}

type DeletePayload struct{
	ID string `json:"id"`
}

func NewService(casher Casher, publisher Publisher, repo Repository, timeout time.Duration) *Service {
	return &Service{
		casher: casher,
		publisher: publisher,
		repository: repo,
		timeout: timeout,
	}
}

func (s *Service) getcontext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), s.timeout)
}

func (s *Service) AddAnswer(answer *entity.Answer) error {
	if answer == nil{
		return errors.New("ansewer can not be nil")
	}

	if err := s.repository.AddAnswer(answer);err != nil{
		return err
	}

	var wg sync.WaitGroup

	errChan := make(chan error, 2)

	wg.Add(1)
	go func ()  {
		defer wg.Done()
		ctx, cancel := s.getcontext()
		defer cancel()
		
		if err := retrier.Do(DefaultRetrierAttempts, DefaultRetryDelay, func() error {
			return s.casher.DoCashing(ctx, fmt.Sprintf(AnswerKeyTemplate, answer.ID.String()), answer)
		});err != nil{
			errChan <- err
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		
		if err := retrier.Do(DefaultRetrierAttempts, DefaultRetryDelay, func() error {
			return s.publisher.Publish(answer, "answer.created")
		});err != nil{
			errChan <- err
		}		
	}()

	wg.Wait()
	close(errChan)

	for err := range errChan{
		return err		
	}

	return nil
}

func (s *Service) Delete(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil{
		return err
	}

	if err = s.repository.DeleteAnswer(uid);err != nil{
		return err
	}

	errChan := make(chan error, 2)

	var wg sync.WaitGroup

	wg.Add(1)
	go func ()  {
		defer wg.Done()
		ctx, cancel := s.getcontext()
		defer cancel()
		
		if err = retrier.Do(DefaultRetrierAttempts, DefaultRetryDelay, func() error {
			return s.casher.DeleteFromCash(ctx, id)
		});err != nil{
			errChan <- err
		}
	}()

	wg.Add(1)
	go func ()  {
		defer wg.Done()
		
		if err = retrier.Do(DefaultRetrierAttempts, DefaultRetryDelay, func() error {
			return s.publisher.Publish(&DeletePayload{
				ID: id,
			}, "answer.deleted")
		});err != nil{
			errChan <- err
		}
	}()

	for err = range errChan{
		return err
	}

	close(errChan)

	return nil
}