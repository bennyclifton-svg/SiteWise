package httpapi

import (
	"context"

	"sitewise/internal/reports"
	"sitewise/internal/store"
)

// AttentionForTest is reports.Attention over one project's records, loaded
// as chat's attention read loads them.
func AttentionForTest(ctx context.Context, st *store.Store, org, project string) ([]reports.AttentionItem, error) {
	in, err := loadAttention(ctx, Deps{Store: st}, opActor{OrgID: org}, project, attentionAll)
	return reports.Attention(in), err
}

// FilterRowsForTest is the rows one named filter returns.
func FilterRowsForTest(ctx context.Context, st *store.Store, org, project, table, name string) ([]string, error) {
	return viewFilters[table][name].rows(ctx, Deps{Store: st}, opActor{OrgID: org}, project)
}
