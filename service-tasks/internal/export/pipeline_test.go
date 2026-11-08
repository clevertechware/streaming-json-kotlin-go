package export

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunInsertsEveryRowOfTheStreamInBatches(t *testing.T) {
	var batchCount, rowCount int
	insert := func(_ context.Context, batch []Movement) error {
		batchCount++
		rowCount += len(batch)
		return nil
	}

	inserted, err := Run(context.Background(), strings.NewReader(streamOf(1050)), insert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inserted != 1050 || rowCount != 1050 || batchCount != 11 {
		t.Errorf("inserted=%d rows=%d batches=%d, want 1050 rows in 11 batches", inserted, rowCount, batchCount)
	}
}

func TestRunStopsReadingAndReturnsTheInsertErrorWhenAnInsertFails(t *testing.T) {
	boom := errors.New("insert refused")
	var calls atomic.Int32
	insert := func(_ context.Context, _ []Movement) error {
		if calls.Add(1) == 2 {
			return boom
		}
		return nil
	}
	finished := make(chan error, 1)

	go func() {
		_, err := Run(context.Background(), strings.NewReader(streamOf(50_000)), insert)
		finished <- err
	}()

	select {
	case err := <-finished:
		if !errors.Is(err, boom) {
			t.Errorf("error = %v, want %v", err, boom)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return: a goroutine is blocked on the channel")
	}
}

func TestRunReturnsAnErrorAndTheRowsAlreadyInsertedWhenTheStreamIsTruncated(t *testing.T) {
	full := streamOf(250)
	var rows int
	insert := func(_ context.Context, batch []Movement) error {
		rows += len(batch)
		return nil
	}

	inserted, err := Run(context.Background(), strings.NewReader(full[:len(full)-20]), insert)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if inserted != rows || inserted != 200 {
		t.Errorf("inserted=%d counted=%d, want 200 rows from the two full batches", inserted, rows)
	}
}

func producer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/movements/export" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestOpenStreamReturnsTheBodyWhenTheProducerAnswers200(t *testing.T) {
	server := producer(t, http.StatusOK, streamOf(120))
	body, err := OpenStream(context.Background(), NewHTTPClient(), server.URL, "WH-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer body.Close()

	inserted, err := Run(context.Background(), body, func(context.Context, []Movement) error { return nil })
	if err != nil || inserted != 120 {
		t.Errorf("inserted=%d err=%v, want 120 rows and no error", inserted, err)
	}
}

func TestOpenStreamFailsBeforeReadingTheBodyWhenStatusIsNot200(t *testing.T) {
	server := producer(t, http.StatusInternalServerError, `{"error":"database down"}`)

	_, err := OpenStream(context.Background(), NewHTTPClient(), server.URL, "WH-1")

	if err == nil || !strings.Contains(err.Error(), "500") || !strings.Contains(err.Error(), "database down") {
		t.Errorf("error = %v, want status 500 and the body excerpt", err)
	}
}

func TestOpenStreamFailsWhenTheContextIsAlreadyCancelled(t *testing.T) {
	server := producer(t, http.StatusOK, streamOf(1))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := OpenStream(ctx, NewHTTPClient(), server.URL, "WH-1"); !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}

func TestNewHTTPClientHasNoGlobalTimeoutAndWaitsNinetySecondsForHeaders(t *testing.T) {
	client := NewHTTPClient()

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", client.Transport)
	}
	if client.Timeout != 0 || transport.ResponseHeaderTimeout != 90*time.Second {
		t.Errorf("Timeout=%v ResponseHeaderTimeout=%v, want 0 and 90s", client.Timeout, transport.ResponseHeaderTimeout)
	}
}

type generatingBody struct {
	total   int
	emitted atomic.Int64
	pending []byte
	closed  bool
}

func (g *generatingBody) Read(p []byte) (int, error) {
	if len(g.pending) == 0 {
		g.pending = g.next()
	}
	if len(g.pending) == 0 {
		return 0, io.EOF
	}
	n := copy(p, g.pending)
	g.pending = g.pending[n:]
	return n, nil
}

func (g *generatingBody) next() []byte {
	index := int(g.emitted.Load())
	switch {
	case g.closed:
		return nil
	case index == 0:
		g.emitted.Add(1)
		return []byte(`{"results":[` + objectOf(0))
	case index < g.total:
		g.emitted.Add(1)
		return []byte("," + objectOf(index))
	default:
		g.closed = true
		return []byte("]}")
	}
}

func objectOf(index int) string {
	return fmt.Sprintf(`{"id":"MV-%05d","warehouseId":"WH-1","code":"C-%05d","label":"Mouvement %d"}`, index, index, index)
}

func TestRunReadsNoMoreThanTheChannelCapacityPlusTwoBatchesWhileInsertIsBlocked(t *testing.T) {
	const articleChannelCapacity, decoderBufferMargin = 4, 20
	maxObjects := (articleChannelCapacity+2)*batchSize + decoderBufferMargin
	body := &generatingBody{total: 50_000}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	insertStarted := make(chan struct{}, 1)
	insert := func(ctx context.Context, _ []Movement) error {
		select {
		case insertStarted <- struct{}{}:
		default:
		}
		<-ctx.Done()
		return ctx.Err()
	}
	finished := make(chan error, 1)
	go func() {
		_, err := Run(ctx, body, insert)
		finished <- err
	}()

	<-insertStarted
	previous := int64(-1)
	for body.emitted.Load() != previous {
		previous = body.emitted.Load()
		time.Sleep(200 * time.Millisecond)
	}
	readWhileBlocked := body.emitted.Load()
	cancel()

	if err := <-finished; !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
	if readWhileBlocked > int64(maxObjects) {
		t.Errorf("objects read while insert is blocked = %d, want at most %d", readWhileBlocked, maxObjects)
	}
}
