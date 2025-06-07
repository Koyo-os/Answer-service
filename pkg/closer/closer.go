package closer

import (
	"context"

	"github.com/Koyo-os/answer-service/pkg/logger"
	"go.uber.org/zap"
)

type (
	Closer interface{
		Close() error
	}

	Shutdown struct{
		closers []Closer
		logger *logger.Logger
	}
)

func NewShutdown(closers ...Closer) *Shutdown {
	return &Shutdown{
		closers: closers,
		logger: logger.Get(),
	}
}

func (s *Shutdown) ShutdownAll(ctx context.Context) {
	for _, closer := range s.closers{
		if err := closer.Close();err != nil{
			s.logger.Error("error close", zap.Error(err))
		}
	}
}