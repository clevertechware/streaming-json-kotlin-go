package export

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestInsertStatementNumbersFourPlaceholdersPerRow(t *testing.T) {
	batch := []Movement{{ID: "a", Code: "c1", Label: "l1"}, {ID: "b", Code: "c2", Label: "l2"}}

	query, args := insertStatement("TP-1", batch)

	if !strings.HasSuffix(query, "VALUES ($1,$2,$3,$4),($5,$6,$7,$8)") {
		t.Errorf("query = %s", query)
	}
	want := []any{"TP-1", "a", "c1", "l1", "TP-1", "b", "c2", "l2"}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("args = %v, want %v", args, want)
		}
	}
}

func newStore(t *testing.T) *Store {
	t.Helper()
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("tasks"),
		postgres.WithUsername("demo"),
		postgres.WithPassword("demo"),
		postgres.BasicWaitStrategies(),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Errorf("terminate postgres container: %v", err)
		}
	})
	url, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	store, err := NewStore(ctx, url)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(store.Close)
	return store
}

func TestStoreInsertsTheWholeStreamIntoPostgresAndRestoresLabels(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	label := "Palette \"fragile\" \\ n°1"
	movements := []Movement{{ID: "special", WarehouseID: "S", Code: "C", Label: label}}
	for i := 0; i < 249; i++ {
		movements = append(movements, Movement{ID: fmt.Sprintf("MV-%d", i), WarehouseID: "S", Code: "C", Label: "L"})
	}
	body := `{"results":` + mustMarshal(movements) + `}`

	inserted, err := Run(ctx, strings.NewReader(body), store.InsertFunc("TP-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	count, err := store.Count(ctx, "TP-1")
	if err != nil || count != 250 || inserted != 250 {
		t.Fatalf("count=%d inserted=%d err=%v, want 250 rows", count, inserted, err)
	}
	var stored string
	row := store.pool.QueryRow(ctx, "SELECT label FROM task_plan_movements WHERE movement_id = 'special'")
	if err := row.Scan(&stored); err != nil || stored != label {
		t.Errorf("stored label = %q err=%v, want %q", stored, err, label)
	}
}

func TestStoreResetAllowsRegeneratingATaskPlanThatAlreadyHoldsRows(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	full := streamOf(250)
	if _, err := Run(ctx, strings.NewReader(full), store.InsertFunc("TP-1")); err != nil {
		t.Fatalf("first generation failed: %v", err)
	}
	if _, err := Run(ctx, strings.NewReader(full), store.InsertFunc("TP-1")); err == nil {
		t.Fatal("expected a primary key violation without reset")
	}

	if err := store.Reset(ctx, "TP-1"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if _, err := Run(ctx, strings.NewReader(full), store.InsertFunc("TP-1")); err != nil {
		t.Fatalf("regeneration failed: %v", err)
	}

	if count, err := store.Count(ctx, "TP-1"); err != nil || count != 250 {
		t.Errorf("count=%d err=%v, want 250", count, err)
	}
}
