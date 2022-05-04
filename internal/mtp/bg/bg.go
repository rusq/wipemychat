// Package bg implements wrapper for running client in background.
//
// TODO: Once https://github.com/gotd/contrib/pull/216 is merged can be removed.
package bg

import (
	"context"
	"errors"
)

// Client abstracts telegram client.
type Client interface {
	Run(ctx context.Context, f func(ctx context.Context) error) error
}

// StopFunc closes Client and waits until Run returns.
type StopFunc func() error

type connectOptions struct {
	ctx context.Context
}

// Option for Connect.
type Option interface {
	apply(o *connectOptions)
}

type fnOption func(o *connectOptions)

func (f fnOption) apply(o *connectOptions) {
	f(o)
}

// WithContext sets base context for client.
func WithContext(ctx context.Context) Option {
	return fnOption(func(o *connectOptions) {
		o.ctx = ctx
	})
}

// Connect blocks until client is connected, calling Run internally in
// background.
func Connect(client Client, options ...Option) (StopFunc, error) {
	opt := &connectOptions{
		ctx: context.Background(),
	}
	for _, o := range options {
		o.apply(opt)
	}

	ctx, cancel := context.WithCancel(opt.ctx)

	initDone := make(chan struct{})
	errC := make(chan error, 1)
	go func() {
		defer close(errC)
		errC <- client.Run(ctx, func(ctx context.Context) error {
			close(initDone)
			<-ctx.Done()
			if errors.Is(ctx.Err(), context.Canceled) {
				return nil
			}
			return ctx.Err()
		})
	}()

	select {
	case <-ctx.Done(): // context cancelled
		cancel()
		return func() error { return nil }, ctx.Err()
	case err := <-errC: // startup timeout
		cancel()
		return func() error { return nil }, err
	case <-initDone: // init done
	}

	stopFn := func() error {
		cancel()
		return <-errC
	}
	return stopFn, nil
}
