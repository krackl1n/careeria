package interceptor

import (
	"context"
	"errors"
	"io"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type finishingClientStream struct {
	grpc.ClientStream

	finish                 func(error)
	finishOnSuccessfulRecv bool

	once sync.Once
	done chan struct{}
}

func wrapClientStream(
	stream grpc.ClientStream,
	watchCtx context.Context,
	desc *grpc.StreamDesc,
	finish func(error),
) grpc.ClientStream {
	wrapped := &finishingClientStream{
		ClientStream:           stream,
		finish:                 finish,
		finishOnSuccessfulRecv: desc != nil && desc.ClientStreams && !desc.ServerStreams,
		done:                   make(chan struct{}),
	}

	if watchCtx != nil && watchCtx.Done() != nil {
		go func() {
			select {
			case <-watchCtx.Done():
				wrapped.complete(watchCtx.Err())
			case <-wrapped.done:
			}
		}()
	}

	return wrapped
}

func (s *finishingClientStream) Header() (metadata.MD, error) {
	md, err := s.ClientStream.Header()
	if err != nil {
		s.complete(err)
	}

	return md, err
}

func (s *finishingClientStream) SendMsg(message any) error {
	err := s.ClientStream.SendMsg(message)
	if err != nil && !errors.Is(err, io.EOF) {
		s.complete(err)
	}

	return err
}

func (s *finishingClientStream) RecvMsg(message any) error {
	err := s.ClientStream.RecvMsg(message)
	if err != nil {
		s.complete(err)
		return err
	}

	if s.finishOnSuccessfulRecv {
		s.complete(nil)
	}

	return nil
}

func (s *finishingClientStream) CloseSend() error {
	err := s.ClientStream.CloseSend()
	if err != nil {
		s.complete(err)
	}

	return err
}

func (s *finishingClientStream) complete(err error) {
	s.once.Do(func() {
		close(s.done)

		if s.finish != nil {
			s.finish(err)
		}
	})
}
