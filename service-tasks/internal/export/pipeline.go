package export

import (
	"context"
	"io"

	"golang.org/x/sync/errgroup"
)

const channelCapacity = 4

type InsertFunc func(ctx context.Context, batch []Movement) error

// Run reads the movements from body in batches and hands each batch to insert.
// The first error cancels the other goroutine, so neither one is left blocked on the channel.
func Run(ctx context.Context, body io.Reader, insert InsertFunc) (int, error) {
	group, ctx := errgroup.WithContext(ctx)
	batches := make(chan []Movement, channelCapacity)
	inserted := 0

	group.Go(func() error {
		defer close(batches)
		return readInBatches(ctx, body, batches)
	})
	group.Go(func() error {
		for batch := range batches {
			if err := insert(ctx, batch); err != nil {
				return err
			}
			inserted += len(batch)
		}
		return nil
	})

	err := group.Wait()
	return inserted, err
}
