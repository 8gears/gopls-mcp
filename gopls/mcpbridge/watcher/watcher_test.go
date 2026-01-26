package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"golang.org/x/tools/gopls/internal/protocol"
	"golang.org/x/tools/internal/testenv"
)

// mockServer is a test double that implements protocol.Server for testing.
// It tracks file change notifications so tests can verify the watcher works.
// All other methods return "not implemented" errors.
type mockServer struct {
	mu              sync.Mutex
	fileChangeCount int
	lastFileChanges []protocol.FileEvent
}

func (m *mockServer) DidChangeWatchedFiles(ctx context.Context, params *protocol.DidChangeWatchedFilesParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fileChangeCount++
	m.lastFileChanges = params.Changes
	return nil
}

func (m *mockServer) GetFileChangeCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.fileChangeCount
}

func (m *mockServer) GetLastFileChanges() []protocol.FileEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastFileChanges
}

// Stub implementations for all other protocol.Server methods (abbreviated for brevity)
func (m *mockServer) Progress(context.Context, *protocol.ProgressParams) error {
	return fmt.Errorf("not implemented")
}
func (m *mockServer) SetTrace(context.Context, *protocol.SetTraceParams) error {
	return fmt.Errorf("not implemented")
}
func (m *mockServer) IncomingCalls(context.Context, *protocol.CallHierarchyIncomingCallsParams) ([]protocol.CallHierarchyIncomingCall, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) OutgoingCalls(context.Context, *protocol.CallHierarchyOutgoingCallsParams) ([]protocol.CallHierarchyOutgoingCall, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) ResolveCodeAction(context.Context, *protocol.CodeAction) (*protocol.CodeAction, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) ResolveCodeLens(context.Context, *protocol.CodeLens) (*protocol.CodeLens, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) ResolveCommand(context.Context, *protocol.ExecuteCommandParams) (*protocol.ExecuteCommandParams, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) ResolveCompletionItem(context.Context, *protocol.CompletionItem) (*protocol.CompletionItem, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) ResolveDocumentLink(context.Context, *protocol.DocumentLink) (*protocol.DocumentLink, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) Exit(context.Context) error { return fmt.Errorf("not implemented") }
func (m *mockServer) Initialize(context.Context, *protocol.ParamInitialize) (*protocol.InitializeResult, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) Initialized(context.Context, *protocol.InitializedParams) error {
	return fmt.Errorf("not implemented")
}
func (m *mockServer) Resolve(context.Context, *protocol.InlayHint) (*protocol.InlayHint, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) DidChangeNotebookDocument(context.Context, *protocol.DidChangeNotebookDocumentParams) error {
	return fmt.Errorf("not implemented")
}
func (m *mockServer) DidCloseNotebookDocument(context.Context, *protocol.DidCloseNotebookDocumentParams) error {
	return fmt.Errorf("not implemented")
}
func (m *mockServer) DidOpenNotebookDocument(context.Context, *protocol.DidOpenNotebookDocumentParams) error {
	return fmt.Errorf("not implemented")
}
func (m *mockServer) DidSaveNotebookDocument(context.Context, *protocol.DidSaveNotebookDocumentParams) error {
	return fmt.Errorf("not implemented")
}
func (m *mockServer) Shutdown(context.Context) error { return fmt.Errorf("not implemented") }
func (m *mockServer) CodeAction(context.Context, *protocol.CodeActionParams) ([]protocol.CodeAction, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) CodeLens(context.Context, *protocol.CodeLensParams) ([]protocol.CodeLens, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) ColorPresentation(context.Context, *protocol.ColorPresentationParams) ([]protocol.ColorPresentation, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) Completion(context.Context, *protocol.CompletionParams) (*protocol.CompletionList, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) Declaration(context.Context, *protocol.DeclarationParams) (*protocol.Or_textDocument_declaration, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) Definition(context.Context, *protocol.DefinitionParams) ([]protocol.Location, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) Diagnostic(context.Context, *protocol.DocumentDiagnosticParams) (*protocol.DocumentDiagnosticReport, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) DidChange(context.Context, *protocol.DidChangeTextDocumentParams) error {
	return fmt.Errorf("not implemented")
}
func (m *mockServer) DidClose(context.Context, *protocol.DidCloseTextDocumentParams) error {
	return fmt.Errorf("not implemented")
}
func (m *mockServer) DidOpen(context.Context, *protocol.DidOpenTextDocumentParams) error {
	return fmt.Errorf("not implemented")
}
func (m *mockServer) DidSave(context.Context, *protocol.DidSaveTextDocumentParams) error {
	return fmt.Errorf("not implemented")
}
func (m *mockServer) DocumentColor(context.Context, *protocol.DocumentColorParams) ([]protocol.ColorInformation, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) DocumentHighlight(context.Context, *protocol.DocumentHighlightParams) ([]protocol.DocumentHighlight, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) DocumentLink(context.Context, *protocol.DocumentLinkParams) ([]protocol.DocumentLink, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) DocumentSymbol(context.Context, *protocol.DocumentSymbolParams) ([]any, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) FoldingRange(context.Context, *protocol.FoldingRangeParams) ([]protocol.FoldingRange, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) Formatting(context.Context, *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) Hover(context.Context, *protocol.HoverParams) (*protocol.Hover, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) Implementation(context.Context, *protocol.ImplementationParams) ([]protocol.Location, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) InlayHint(context.Context, *protocol.InlayHintParams) ([]protocol.InlayHint, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) InlineCompletion(context.Context, *protocol.InlineCompletionParams) (*protocol.Or_Result_textDocument_inlineCompletion, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) InlineValue(context.Context, *protocol.InlineValueParams) ([]protocol.InlineValue, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) LinkedEditingRange(context.Context, *protocol.LinkedEditingRangeParams) (*protocol.LinkedEditingRanges, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) Moniker(context.Context, *protocol.MonikerParams) ([]protocol.Moniker, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) OnTypeFormatting(context.Context, *protocol.DocumentOnTypeFormattingParams) ([]protocol.TextEdit, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) PrepareCallHierarchy(context.Context, *protocol.CallHierarchyPrepareParams) ([]protocol.CallHierarchyItem, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) PrepareRename(context.Context, *protocol.PrepareRenameParams) (*protocol.PrepareRenameResult, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) PrepareTypeHierarchy(context.Context, *protocol.TypeHierarchyPrepareParams) ([]protocol.TypeHierarchyItem, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) RangeFormatting(context.Context, *protocol.DocumentRangeFormattingParams) ([]protocol.TextEdit, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) RangesFormatting(context.Context, *protocol.DocumentRangesFormattingParams) ([]protocol.TextEdit, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) References(context.Context, *protocol.ReferenceParams) ([]protocol.Location, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) Rename(context.Context, *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) SelectionRange(context.Context, *protocol.SelectionRangeParams) ([]protocol.SelectionRange, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) SemanticTokensFull(context.Context, *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) SemanticTokensFullDelta(context.Context, *protocol.SemanticTokensDeltaParams) (any, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) SemanticTokensRange(context.Context, *protocol.SemanticTokensRangeParams) (*protocol.SemanticTokens, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) SignatureHelp(context.Context, *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) TypeDefinition(context.Context, *protocol.TypeDefinitionParams) ([]protocol.Location, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) WillSave(context.Context, *protocol.WillSaveTextDocumentParams) error {
	return fmt.Errorf("not implemented")
}
func (m *mockServer) WillSaveWaitUntil(context.Context, *protocol.WillSaveTextDocumentParams) ([]protocol.TextEdit, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) Subtypes(context.Context, *protocol.TypeHierarchySubtypesParams) ([]protocol.TypeHierarchyItem, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) Supertypes(context.Context, *protocol.TypeHierarchySupertypesParams) ([]protocol.TypeHierarchyItem, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) WorkDoneProgressCancel(context.Context, *protocol.WorkDoneProgressCancelParams) error {
	return fmt.Errorf("not implemented")
}
func (m *mockServer) DiagnosticWorkspace(context.Context, *protocol.WorkspaceDiagnosticParams) (*protocol.WorkspaceDiagnosticReport, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) DidChangeConfiguration(context.Context, *protocol.DidChangeConfigurationParams) error {
	return fmt.Errorf("not implemented")
}
func (m *mockServer) DidChangeWorkspaceFolders(context.Context, *protocol.DidChangeWorkspaceFoldersParams) error {
	return fmt.Errorf("not implemented")
}
func (m *mockServer) DidCreateFiles(context.Context, *protocol.CreateFilesParams) error {
	return fmt.Errorf("not implemented")
}
func (m *mockServer) DidDeleteFiles(context.Context, *protocol.DeleteFilesParams) error {
	return fmt.Errorf("not implemented")
}
func (m *mockServer) DidRenameFiles(context.Context, *protocol.RenameFilesParams) error {
	return fmt.Errorf("not implemented")
}
func (m *mockServer) ExecuteCommand(context.Context, *protocol.ExecuteCommandParams) (any, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) TextDocumentContent(context.Context, *protocol.TextDocumentContentParams) (*protocol.TextDocumentContentResult, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) WillCreateFiles(context.Context, *protocol.CreateFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) WillDeleteFiles(context.Context, *protocol.DeleteFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) WillRenameFiles(context.Context, *protocol.RenameFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) ResolveWorkspaceSymbol(context.Context, *protocol.WorkspaceSymbol) (*protocol.WorkspaceSymbol, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockServer) Symbol(context.Context, *protocol.WorkspaceSymbolParams) ([]protocol.SymbolInformation, error) {
	return nil, fmt.Errorf("not implemented")
}

// TestWatcherFileChanges tests that the watcher detects file changes and notifies the server.
func TestWatcherFileChanges(t *testing.T) {
	testenv.NeedsExec(t)

	// Create a temporary directory with a simple Go file
	tmpDir := t.TempDir()
	mainGo := filepath.Join(tmpDir, "main.go")
	content := `package main

func Hello() string {
	return "hello"
}

func main() {}
`
	if err := os.WriteFile(mainGo, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}

	// Create mock server
	mock := &mockServer{}

	// Create watcher with mock server
	w, err := New(mock, tmpDir)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer w.Close()

	// Give the watcher time to start watching
	time.Sleep(100 * time.Millisecond)

	// Modify the file
	t.Log("Modifying main.go...")
	newContent := content + `
func Goodbye() string {
	return "goodbye"
}
`
	if err := os.WriteFile(mainGo, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to modify main.go: %v", err)
	}

	// Wait for watcher to detect and process the change
	// The watcher batches events with 500ms delay
	time.Sleep(1 * time.Second)

	// Verify the mock server received the file change notification
	changeCount := mock.GetFileChangeCount()
	if changeCount == 0 {
		t.Errorf("Expected at least 1 file change notification, got %d", changeCount)
	} else {
		t.Logf("✓ Watcher detected and notified of file changes (count: %d)", changeCount)

		// Verify the file change event
		changes := mock.GetLastFileChanges()
		if len(changes) > 0 {
			t.Logf("✓ File change event: %s %s", changes[0].Type, changes[0].URI.Path())
		}
	}
}
