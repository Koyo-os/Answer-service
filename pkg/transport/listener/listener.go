package listener

import (
	"context"
	"fmt"

	"github.com/Koyo-os/answer-service/internal/entity"
	"github.com/Koyo-os/answer-service/internal/service"
	"github.com/Koyo-os/answer-service/pkg/logger"
	"github.com/bytedance/sonic"
	"go.uber.org/zap"
)

const (
	// Event types for answer operations
	EventTypeAnswerCreate = "request.answer.create"
	EventTypeAnswerDelete = "request.answer.delete"

	// Channel buffer size for events
	DefaultEventChannelSize = 100
)

// Listener handles incoming events and processes them accordingly.
// It acts as an event-driven processor for answer-related operations.
type Listener struct {
	logger  *logger.Logger
	service *service.Service
	events  chan entity.Event
}

// NewListener creates a new Listener instance with the provided dependencies.
// It initializes the event channel with a default buffer size to prevent blocking.
func NewListener(logger *logger.Logger, service *service.Service, events chan entity.Event) *Listener {
	return &Listener{
		logger:  logger,
		service: service,
		events:  events,
	}
}

// NewListenerWithChannelSize creates a new Listener with a custom event channel buffer size.
// This allows for fine-tuning the event processing capacity based on expected load.
func NewListenerWithChannelSize(logger *logger.Logger, service *service.Service, channelSize int) *Listener {
	return &Listener{
		logger:  logger,
		service: service,
		events:  make(chan entity.Event, channelSize),
	}
}

// SendEvent sends an event to the listener's event channel.
// It returns an error if the channel is full to prevent blocking.
func (l *Listener) SendEvent(event entity.Event) error {
	select {
	case l.events <- event:
		return nil
	default:
		return fmt.Errorf("event channel is full, dropping event: %s", event.ID)
	}
}

// Run starts the event listener loop and processes incoming events.
// It runs until the context is cancelled, ensuring graceful shutdown.
func (l *Listener) Run(ctx context.Context) {
	l.logger.Info("starting event listener")

	defer func() {
		l.logger.Info("event listener stopped")
		close(l.events)
	}()

	for {
		select {
		case event := <-l.events:
			l.processEvent(ctx, event)
		case <-ctx.Done():
			l.logger.Info("received shutdown signal, stopping event listener")
			return
		}
	}
}

// processEvent handles individual event processing based on event type.
// It delegates to specific handler methods for better code organization.
func (l *Listener) processEvent(ctx context.Context, event entity.Event) {
	l.logger.Debug("processing event",
		zap.String("event_id", event.ID),
		zap.String("event_type", event.Type))

	switch event.Type {
	case EventTypeAnswerCreate:
		l.handleAnswerCreate(event)
	case EventTypeAnswerDelete:
		l.handleAnswerDelete(event)
	default:
		l.logger.Warn("unknown event type received",
			zap.String("event_id", event.ID),
			zap.String("event_type", event.Type))
	}
}

// handleAnswerCreate processes answer creation events.
// It unmarshals the event payload and delegates to the service layer.
func (l *Listener) handleAnswerCreate(event entity.Event) {
	answer := new(entity.Answer)

	// Unmarshal the event payload into an Answer entity
	if err := sonic.Unmarshal(event.Payload, answer); err != nil {
		l.logger.Error("failed to unmarshal answer creation event payload",
			zap.String("event_id", event.ID),
			zap.String("event_type", event.Type),
			zap.Error(err))
		return
	}

	// Validate the unmarshaled answer
	if err := l.validateAnswer(answer); err != nil {
		l.logger.Error("invalid answer data in creation event",
			zap.String("event_id", event.ID),
			zap.String("answer_id", answer.ID.String()),
			zap.Error(err))
		return
	}

	// Process the answer creation through the service layer
	if err := l.service.Add(answer); err != nil {
		l.logger.Error("failed to add answer",
			zap.String("event_id", event.ID),
			zap.String("answer_id", answer.ID.String()),
			zap.Error(err))
		return
	}

	l.logger.Info("successfully processed answer creation event",
		zap.String("event_id", event.ID),
		zap.String("answer_id", answer.ID.String()))
}

// handleAnswerDelete processes answer deletion events.
// It unmarshals the event payload and delegates to the service layer.
func (l *Listener) handleAnswerDelete(event entity.Event) {
	// Define a struct for the delete request payload
	req := &struct {
		ID string `json:"id" validate:"required,uuid"`
	}{}

	// Unmarshal the event payload into the delete request struct
	if err := sonic.Unmarshal(event.Payload, req); err != nil {
		l.logger.Error("failed to unmarshal answer deletion event payload",
			zap.String("event_id", event.ID),
			zap.String("event_type", event.Type),
			zap.Error(err))
		return
	}

	// Validate the request data
	if req.ID == "" {
		l.logger.Error("missing answer ID in deletion event",
			zap.String("event_id", event.ID))
		return
	}

	// Process the answer deletion through the service layer
	if err := l.service.Delete(req.ID); err != nil {
		l.logger.Error("failed to delete answer",
			zap.String("event_id", event.ID),
			zap.String("answer_id", req.ID),
			zap.Error(err))
		return
	}

	l.logger.Info("successfully processed answer deletion event",
		zap.String("event_id", event.ID),
		zap.String("answer_id", req.ID))
}

// validateAnswer performs basic validation on the answer entity.
// This helps catch invalid data early in the processing pipeline.
func (l *Listener) validateAnswer(answer *entity.Answer) error {
	if answer == nil {
		return fmt.Errorf("answer is nil")
	}

	if answer.ID.String() == "" {
		return fmt.Errorf("answer ID is empty")
	}

	return nil
}

// GetEventChannelLength returns the current number of events in the channel.
// This can be useful for monitoring and debugging purposes.
func (l *Listener) GetEventChannelLength() int {
	return len(l.events)
}

// GetEventChannelCapacity returns the total capacity of the event channel.
func (l *Listener) GetEventChannelCapacity() int {
	return cap(l.events)
}
