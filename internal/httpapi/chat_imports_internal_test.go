package httpapi

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The owner's condition: no screen read or write waits on the model. Chat
// is the only path that does. This test holds it by the import graph, so it
// fails the build rather than relying on anyone remembering.
//
//  1. Outside internal/httpapi, only the packages named below may import
//     the agent loop or the model client.
//  2. Inside internal/httpapi, only the chat files may import the agent
//     loop, and only they and the tender pass's files may import the model
//     client or touch Deps.Chat.
func TestOnlyChatReachesTheModel(t *testing.T) {
	const agentPath, llmPath = `"sitewise/internal/agent"`, `"sitewise/internal/llm"`
	// Packages that are the model path itself. tender is rule 1's
	// background job for quotes and runs in the worker, not in a request.
	mayUseModel := map[string]bool{"agent": true, "llm": true, "tender": true, "httpapi": true}
	root := filepath.Join("..", "..", "internal")
	dirs, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	imports := func(path string) map[string]bool {
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		out := map[string]bool{}
		for _, im := range f.Imports {
			out[im.Path.Value] = true
		}
		return out
	}
	for _, dir := range dirs {
		if !dir.IsDir() || mayUseModel[dir.Name()] {
			continue
		}
		_ = filepath.WalkDir(filepath.Join(root, dir.Name()), func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			if im := imports(path); im[agentPath] || im[llmPath] {
				t.Errorf("%s imports the model path; only chat may wait on the model", path)
			}
			return nil
		})
	}
	// Inside this package.
	agentFiles := map[string]bool{"chat.go": true, "chat_tools.go": true, "chat_read.go": true, "chat_docs.go": true, "chat_calc.go": true, "chat_prefetch.go": true, "chat_view.go": true, "chat_undo.go": true}
	llmFiles := map[string]bool{"chat.go": true, "chat_tools.go": true, "chat_prefetch.go": true, "server.go": true, "tender_reader.go": true}
	chatFiles := map[string]bool{"chat.go": true, "chat_docs.go": true, "projects.go": true, "server.go": true, "tender_reader.go": true}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		im := imports(name)
		if im[agentPath] && !agentFiles[name] {
			t.Errorf("%s imports the agent loop; only the chat files may", name)
		}
		if im[llmPath] && !llmFiles[name] {
			t.Errorf("%s imports the model client; only chat and the tender pass may", name)
		}
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		src := string(raw)
		if (strings.Contains(src, ".Chat.") || strings.Contains(src, ".Chat ==") || strings.Contains(src, ".Chat !=")) && !chatFiles[name] {
			t.Errorf("%s touches Deps.Chat; a screen handler must not reach the model", name)
		}
	}
	// The operations call is a screen-side write: it must not reach chat.
	for _, name := range []string{"ops.go", "ops_costs.go", "ops_profile.go", "ops_program.go", "ops_tender.go"} {
		if im := imports(name); im[agentPath] || im[llmPath] {
			t.Errorf("%s reaches the model path", name)
		}
	}
}
