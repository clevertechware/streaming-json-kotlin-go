package export

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const batchSize = 100

type Movement struct {
	ID          string `json:"id"`
	WarehouseID string `json:"warehouseId"`
	Code        string `json:"code"`
	Label       string `json:"label"`
}

// The last batch is sent only once the closing "]}" has been read, so a truncated
// stream never produces a partial final batch.
func readInBatches(ctx context.Context, body io.Reader, batches chan<- []Movement) error {
	dec := json.NewDecoder(body)
	if err := skipToResults(dec); err != nil {
		return err
	}
	batch := make([]Movement, 0, batchSize)
	for dec.More() {
		movement, err := decodeMovement(dec)
		if err != nil {
			return err
		}
		batch = append(batch, movement)
		if len(batch) == batchSize {
			if err = send(ctx, batches, batch); err != nil {
				return err
			}
			batch = make([]Movement, 0, batchSize)
		}
	}
	if err := expectDelim(dec, ']'); err != nil {
		return err
	}
	if err := expectDelim(dec, '}'); err != nil {
		return err
	}
	if len(batch) > 0 {
		return send(ctx, batches, batch)
	}
	return nil
}

func send(ctx context.Context, batches chan<- []Movement, batch []Movement) error {
	select {
	case batches <- batch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func skipToResults(dec *json.Decoder) error {
	if err := expectDelim(dec, '{'); err != nil {
		return err
	}
	for dec.More() {
		key, err := dec.Token()
		if err != nil {
			return unexpectedEOF(err)
		}
		if key == "results" {
			return expectDelim(dec, '[')
		}
		var ignored json.RawMessage
		if err = dec.Decode(&ignored); err != nil {
			return unexpectedEOF(err)
		}
	}
	return errors.New("missing \"results\" array")
}

func decodeMovement(dec *json.Decoder) (Movement, error) {
	var movement Movement
	if err := dec.Decode(&movement); err != nil {
		return Movement{}, fmt.Errorf("decode movement: %w", unexpectedEOF(err))
	}
	return movement, nil
}

func expectDelim(dec *json.Decoder, want json.Delim) error {
	token, err := dec.Token()
	if err != nil {
		return fmt.Errorf("expected %q: %w", rune(want), unexpectedEOF(err))
	}
	if got, ok := token.(json.Delim); !ok || got != want {
		return fmt.Errorf("expected %q, got %v", rune(want), token)
	}
	return nil
}

func unexpectedEOF(err error) error {
	if errors.Is(err, io.EOF) {
		return io.ErrUnexpectedEOF
	}
	return err
}
