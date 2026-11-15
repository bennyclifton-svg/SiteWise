package httpapi

import (
	"context"
	"encoding/json"

	"sitewise/internal/agent"
)

// Placeholders until the document tools are written (order of work, step 6).

func (b *chatBox) find(ctx context.Context, input json.RawMessage) agent.Output {
	return agent.Output{Text: "find is not available yet; use the outline and read."}
}

func (b *chatBox) documents(ctx context.Context, input json.RawMessage) agent.Output {
	return agent.Output{Text: "The document tools are not available yet."}
}

func (b *chatBox) passages(ctx context.Context, input json.RawMessage) agent.Output {
	return agent.Output{Text: "The document tools are not available yet."}
}

func readTender(ctx context.Context, d Deps, who opActor, project, pkg string, r *reader) error {
	r.line("Tender lines cannot be read from chat yet.")
	return nil
}

func readMore(ctx context.Context, d Deps, who opActor, project, area string, r *reader) error {
	r.line("%s cannot be read from chat yet.", area)
	return nil
}
