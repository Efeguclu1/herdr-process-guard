package store

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Efeguclu1/herdr-process-guard/internal/model"
)

func TestConcurrentUpdatesAreSerializedAndStateIsPrivate(t *testing.T) {
	directory := t.TempDir()
	storage := New(directory)
	var group sync.WaitGroup
	for index := 0; index < 12; index++ {
		index := index
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := storage.Update(func(state *model.PersistedState) error {
				key := model.ProcessKey{PID: index + 10, StartUnixMS: 1000}.String()
				state.Observations[key] = model.Observation{Key: model.ProcessKey{PID: index + 10, StartUnixMS: 1000}, CommandSummary: "node", CommandHash: fmt.Sprint(index), FirstObservedAt: time.Now(), LastObservedAt: time.Now()}
				return nil
			})
			if err != nil {
				t.Errorf("update: %v", err)
			}
		}()
	}
	group.Wait()
	state, err := storage.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Observations) != 12 {
		t.Fatalf("lost concurrent updates: %d", len(state.Observations))
	}
	bytes, err := os.ReadFile(storage.statePath())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(bytes), "raw-secret") {
		t.Fatal("raw secret persisted")
	}
}

func TestRejectsUnknownSchema(t *testing.T) {
	storage := New(t.TempDir())
	if err := os.WriteFile(storage.statePath(), []byte(`{"schema_version":99,"observations":{},"approvals":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.Load(); err == nil {
		t.Fatal("expected unsupported schema error")
	}
}
