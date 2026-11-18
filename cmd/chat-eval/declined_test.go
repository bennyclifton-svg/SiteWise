package main

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"sitewise/internal/auth"
	"sitewise/internal/store"
)

// Owner decision 3: the report counts every org's turns and lists the
// words of declined messages for the named org only. Another org's words
// are never printed, with or without -org.
func TestDeclinedReportListsOneOrgsWords(t *testing.T) {
	dsn := os.Getenv("SITEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("SITEWISE_TEST_DATABASE_URL is required")
	}
	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	since := time.Now().Add(-time.Minute)
	org := func(words ...string) string {
		t.Helper()
		_, id, err := auth.Bootstrap(ctx, st, &auth.Sink{}, uuid(), uuid()+"@example.test", nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = st.DeleteOrg(context.Background(), id) })
		project := uuid()
		if err := st.CreateProject(ctx, id, project, "Declines"); err != nil {
			t.Fatal(err)
		}
		for i, w := range words {
			turn, err := st.StartChatTurn(ctx, id, project, uuid(), "", w)
			if err != nil {
				t.Fatal(err)
			}
			turn.Status, turn.Declined = "done", i%2 == 0
			if err := st.FinishChatTurn(ctx, id, turn, nil); err != nil {
				t.Fatal(err)
			}
		}
		return id
	}
	owner := org("Owner asks for the messages table", "Owner adds a line")
	org("Stranger's private wording about RFIs", "Stranger adds a line")

	var withOrg, without bytes.Buffer
	if err := reportDeclined(ctx, st, owner, since, &withOrg); err != nil {
		t.Fatal(err)
	}
	if err := reportDeclined(ctx, st, "", since, &without); err != nil {
		t.Fatal(err)
	}
	for name, out := range map[string]string{"with -org": withOrg.String(), "without -org": without.String()} {
		if strings.Contains(out, "Stranger") {
			t.Errorf("%s: another org's words were printed:\n%s", name, out)
		}
		if !strings.Contains(out, "| Week from | Turns | Declined | Share |") || !strings.Contains(out, "turns declined") {
			t.Errorf("%s: no weekly counts:\n%s", name, out)
		}
	}
	if !strings.Contains(withOrg.String(), "Owner asks for the messages table") || strings.Contains(withOrg.String(), "Owner adds a line") {
		t.Errorf("the owner's declined message, and only it, is listed:\n%s", withOrg.String())
	}
	if strings.Contains(without.String(), "Owner asks") {
		t.Errorf("without -org no message is listed:\n%s", without.String())
	}
	if err := reportDeclined(ctx, st, "not-an-org", since, &bytes.Buffer{}); err == nil {
		t.Error("an -org that is not an id must be refused")
	}
}
