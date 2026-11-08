package export

import (
	"context"
	"io"
	"strings"
	"testing"
)

func collect(t *testing.T, body string) ([][]Movement, error) {
	t.Helper()
	batches := make(chan []Movement, 1000)
	err := readInBatches(context.Background(), strings.NewReader(body), batches)
	close(batches)
	var collected [][]Movement
	for batch := range batches {
		collected = append(collected, batch)
	}
	return collected, err
}

func TestReadInBatchesSplitsTheStreamIntoBatchesOfOneHundred(t *testing.T) {
	tests := []struct {
		name      string
		count     int
		wantSizes []int
	}{
		{"empty results array", 0, nil},
		{"fewer rows than a batch", 7, []int{7}},
		{"exactly one batch", 100, []int{100}},
		{"exact multiple of the batch size", 200, []int{100, 100}},
		{"remainder in a last partial batch", 250, []int{100, 100, 50}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			batches, err := collect(t, streamOf(tc.count))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var sizes []int
			total := 0
			for _, batch := range batches {
				sizes = append(sizes, len(batch))
				total += len(batch)
			}
			if total != tc.count {
				t.Errorf("rows = %d, want %d", total, tc.count)
			}
			if len(sizes) != len(tc.wantSizes) {
				t.Fatalf("batch sizes = %v, want %v", sizes, tc.wantSizes)
			}
			for i := range sizes {
				if sizes[i] != tc.wantSizes[i] {
					t.Errorf("batch sizes = %v, want %v", sizes, tc.wantSizes)
				}
			}
		})
	}
}

func TestReadInBatchesKeepsRowOrderAcrossBatches(t *testing.T) {
	batches, err := collect(t, streamOf(150))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := batches[1][0].ID; got != "MV-00100" {
		t.Errorf("first id of second batch = %q, want MV-00100", got)
	}
}

func TestReadInBatchesRestoresEscapedCharactersInLabels(t *testing.T) {
	label := "Palette \"fragile\" \\ n°1\nAllée A | Rayon 1\t\u0001 é 😀"
	body := `{"results":[` + mustMarshal(Movement{ID: "1", WarehouseID: "S", Code: "C", Label: label}) + `]}`

	batches, err := collect(t, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := batches[0][0].Label; got != label {
		t.Errorf("label = %q, want %q", got, label)
	}
}

func TestReadInBatchesIgnoresUnknownKeysAroundAndInsideResults(t *testing.T) {
	body := `{"meta":{"a":[1,2,{"b":null}]},"results":[{"id":"1","warehouseId":"S","code":"C","label":"L","extra":{"x":1}}]}`

	batches, err := collect(t, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(batches) != 1 || batches[0][0].ID != "1" {
		t.Errorf("batches = %+v, want one batch holding id 1", batches)
	}
}

func TestReadInBatchesFailsOnTruncatedOrInvalidStreams(t *testing.T) {
	full := streamOf(3)
	tests := []struct {
		name string
		body string
	}{
		{"cut in the middle of an object", full[:len(full)-30]},
		{"cut between two objects", full[:strings.Index(full, "},{")+2]},
		{"cut before the closing array", strings.TrimSuffix(full, "]}")},
		{"cut before the closing object", strings.TrimSuffix(full, "}")},
		{"cut before the results array", `{"results":`},
		{"empty body", ``},
		{"results key absent", `{"other":[]}`},
		{"results is not an array", `{"results":{}}`},
		{"body is not an object", `[1,2]`},
		{"malformed object in the array", `{"results":[{"id":1}]}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := collect(t, tc.body); err == nil {
				t.Error("expected an error, got nil")
			}
		})
	}
}

func TestReadInBatchesReportsUnexpectedEOFOnTruncatedStream(t *testing.T) {
	full := streamOf(250)

	_, err := collect(t, full[:len(full)/2])
	if err == nil || !strings.Contains(err.Error(), io.ErrUnexpectedEOF.Error()) {
		t.Errorf("error = %v, want %v", err, io.ErrUnexpectedEOF)
	}
}

func TestReadInBatchesDoesNotSendTheLastPartialBatchWhenStreamIsTruncated(t *testing.T) {
	full := streamOf(250)

	batches, err := collect(t, strings.TrimSuffix(full, "]}"))
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if len(batches) != 2 {
		t.Errorf("batches sent = %d, want 2 full batches only", len(batches))
	}
}

func TestReadInBatchesReturnsWhenContextIsCancelledWhileChannelIsFull(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	batches := make(chan []Movement)
	done := make(chan error, 1)
	go func() { done <- readInBatches(ctx, strings.NewReader(streamOf(500)), batches) }()

	cancel()

	if err := <-done; err != context.Canceled {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}
