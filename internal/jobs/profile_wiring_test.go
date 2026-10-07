package jobs

import (
	"sitewise/internal/knowledge"
	"sitewise/internal/store"
	"sync"
	"testing"
)

func TestWorkerSharesImmutableProfileWiring(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	w := &Worker{Store: &store.Store{}, Catalog: cat}
	stores := make([]*store.Store, 8)
	var done sync.WaitGroup
	for i := range stores {
		done.Add(1)
		go func() { defer done.Done(); stores[i] = w.profileStore() }()
	}
	done.Wait()
	for _, s := range stores {
		if s == nil || s != stores[0] {
			t.Fatal("worker rebuilt immutable wiring")
		}
	}
}
