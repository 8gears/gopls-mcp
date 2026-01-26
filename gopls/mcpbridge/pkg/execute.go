package pkg

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/gopls/internal/cache"
	"golang.org/x/tools/gopls/internal/file"
	"golang.org/x/tools/gopls/internal/golang"
	"golang.org/x/tools/gopls/internal/protocol"
	"golang.org/x/tools/gopls/internal/settings"
	"golang.org/x/tools/gopls/mcpbridge/core"
	"golang.org/x/tools/gopls/mcpbridge/watcher"
)

const (
	// mcpName is the name of the MCP server.
	mcpName = "gopls-mcp"
	// mcpVersion is the version of the MCP server.
	mcpVersion = "v0.0.1"
)

var (
	// addr is the address to listen on (enables HTTP mode).
	addr = flag.String("addr", "", "Address to listen on (e.g., localhost:8080)")
	// verbose enables verbose logging.
	verbose = flag.Bool("verbose", false, "Enable verbose logging")
	// workdirFlag is the Go project directory to analyze (flag).
	workdirFlag = flag.String("workdir", "", "Path to the Go project directory (default is current directory)")
	// configFlag is the path to the MCP configuration file (optional).
	configFlag = flag.String("config", "", "Path to gopls-mcp configuration file (JSON format)")
	// logfile is the path to a log file for debugging (optional).
	// When set, logs are written to this file even in stdio mode.
	logfile = flag.String("logfile", "", "Path to log file for debugging (writes logs even in stdio mode)")
	// allowDynamicViews enables dynamic view creation for testing purposes only.
	// TEST-ONLY FLAG: This allows the handler to create new gopls views on-demand
	// when a Cwd parameter doesn't match any existing view.
	// WARNING: This is intended for e2e testing only. Normal users should not
	// need this flag, as they typically work with a single project.
	// Production usage: Run one gopls-mcp instance per project directory.
	allowDynamicViews = flag.Bool("allow-dynamic-views", false, "TEST-ONLY: Allow dynamic view creation for multiple workdirs (e2e testing)")
)

func Execute() {
	flag.Parse()

	// Configure logging based on transport mode
	// CRITICAL: In stdio mode, NEVER log to stdout/stderr as it corrupts MCP protocol
	if *addr != "" {
		// HTTP mode: logging to stdout is OK
		if *verbose {
			log.SetFlags(log.LstdFlags | log.Lshortfile)
			log.SetOutput(os.Stdout)
		}
	} else if *logfile != "" {
		// Stdio mode with logfile: write logs to file for debugging
		logFile, err := os.OpenFile(*logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Printf("[gopls-mcp] Failed to open log file %s: %v (logs discarded)", *logfile, err)
			log.SetOutput(io.Discard)
		} else {
			log.SetFlags(log.LstdFlags | log.Lshortfile)
			log.SetOutput(logFile)
			log.Printf("[gopls-mcp] Logging to file: %s", *logfile)
		}
	} else {
		// Stdio mode: ALWAYS discard logs, verbose flag ignored for safety
		log.SetOutput(io.Discard)
	}

	// Set the project directory to analyze
	projectDir := *workdirFlag
	if projectDir == "" {
		if dir, err := os.Getwd(); err == nil {
			projectDir = dir
		} else {
			log.Fatalf("[gopls-mcp] Failed to get working directory: %v", err)
		}
	}

	// Load configuration from file if provided
	var config *core.MCPConfig
	if *configFlag != "" {
		log.Printf("[gopls-mcp] Loading configuration from %s", *configFlag)
		data, err := os.ReadFile(*configFlag)
		if err != nil {
			log.Fatalf("[gopls-mcp] Failed to read config file: %v", err)
		}
		config, err = core.LoadConfig(data)
		if err != nil {
			log.Fatalf("[gopls-mcp] Failed to parse config: %v", err)
		}
		log.Printf("[gopls-mcp] Configuration loaded successfully")
	} else {
		// Use default configuration
		config = core.DefaultConfig()
	}

	// Override workdir from config if set
	if config.Workdir != "" {
		projectDir = config.Workdir
		log.Printf("[gopls-mcp] Using workdir from config: %s", projectDir)
	}

	// Create gopls cache
	ctx := context.Background()
	goplsCache := cache.New(nil)
	session := cache.NewSession(ctx, goplsCache)

	// Initialize gopls options (REQUIRED for view creation)
	// Start with defaults and apply user configuration
	options := settings.DefaultOptions()

	// Apply gopls configuration from MCP config
	// This allows users to set native gopls options like analyses, buildFlags, etc.
	if err := config.ApplyGoplsOptions(options); err != nil {
		log.Printf("[gopls-mcp] Warning: Failed to apply some gopls options: %v", err)
		// Continue anyway - partial configuration is better than failure
	} else {
		log.Printf("[gopls-mcp] Gopls options applied successfully")
	}

	// Fetch Go environment for the working directory (REQUIRED for view creation)
	// This loads GOROOT, GOPATH, GOVERSION, and other critical environment info
	dirURI := protocol.URIFromPath(projectDir)
	goEnv, err := cache.FetchGoEnv(ctx, dirURI, options)
	if err != nil {
		log.Fatalf("[gopls-mcp] Failed to load Go env: %v", err)
	}

	// Create a view for the working directory with proper initialization
	// This enables gopls to analyze the Go project
	folder := &cache.Folder{
		Dir:     dirURI,
		Options: options,
		Env:     *goEnv,
	}
	view, _, releaseView, err := session.NewView(ctx, folder)
	if err != nil {
		log.Fatalf("[gopls-mcp] Failed to create view for %s: %v", projectDir, err)
	}
	defer releaseView()

	log.Printf("[gopls-mcp] Created view for %s (type: %v)", projectDir, view.Type())
	releaseView() // Release the initial snapshot since we won't use it

	// Create a minimal LSP server stub that implements the methods we need
	// The gopls-mcp handlers use the Symbol method for search
	// The watcher uses DidChangeWatchedFiles to notify gopls of file changes
	lspServer := &minimalServer{session: session}

	// Start file change watcher
	// This keeps the gopls cache up-to-date when files are edited
	var fileWatcher *watcher.Watcher
	fileWatcher, err = watcher.New(lspServer, projectDir)
	if err != nil {
		log.Printf("[gopls-mcp] Failed to start file watcher: %v", err)
		// Continue anyway - tools will work but file changes won't be detected
	} else {
		defer fileWatcher.Close()
		log.Printf("[gopls-mcp] File watcher started for %s", projectDir)
	}

	// Create gopls-mcp handler backed by gopls session
	// Pass the allowDynamicViews flag to enable test-only dynamic view creation
	// Pass the config to enable response limits
	var handlerOpts []core.HandlerOption
	handlerOpts = append(handlerOpts, core.WithConfig(config))
	if *allowDynamicViews {
		handlerOpts = append(handlerOpts, core.WithDynamicViews(true))
	}
	coreHandler := core.NewHandler(session, lspServer, handlerOpts...)

	// Create MCP server and register all gopls-mcp tools
	server := mcp.NewServer(&mcp.Implementation{Name: mcpName, Version: mcpVersion}, nil)
	core.RegisterTools(server, coreHandler)

	log.Printf("[gopls-mcp] Registered %d MCP tools for Go analysis", 18)
	log.Printf("[gopls-mcp] Working directory: %s", projectDir)

	if *addr != "" {
		// HTTP mode
		handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
			return server
		}, &mcp.StreamableHTTPOptions{JSONResponse: true})

		http.Handle("/", handler)
		log.Printf("[gopls-mcp] Starting %s HTTP server at %s", mcpName, *addr)
		if err := http.ListenAndServe(*addr, nil); err != nil {
			log.Fatalf("[gopls-mcp] HTTP server failed: %v", err)
		}
		return
	}

	// Stdio mode (default)
	log.Printf("[gopls-mcp] Starting %s in stdio mode", mcpName)

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Run server in a goroutine so we can handle signals
	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.Run(ctx, &mcp.StdioTransport{})
	}()

	// Wait for either server error or signal
	select {
	case err := <-serverErrCh:
		// Server ended (likely connection closed)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[gopls-mcp] Server ended: %v\n", err)
		}
	case sig := <-sigCh:
		// Received signal, exit gracefully
		fmt.Fprintf(os.Stderr, "[gopls-mcp] Received signal: %v\n", sig)
	}
	// Always exit cleanly - stdio mode ends when client closes connection
}

// minimalServer implements protocol.Server with only the methods used by core.
// Most methods return "not implemented" errors, but Symbol is functional.
type minimalServer struct {
	session *cache.Session
}

// Symbol implements workspace symbol search using gopls's internal golang package.
// This is the only method from protocol.Server that gopls-mcp handlers currently use.
func (s *minimalServer) Symbol(ctx context.Context, params *protocol.WorkspaceSymbolParams) ([]protocol.SymbolInformation, error) {
	// Collect ALL snapshots from ALL views
	// This is critical for multi-module workspaces where each module has its own view
	views := s.session.Views()
	if len(views) == 0 {
		return nil, fmt.Errorf("no active views")
	}

	snapshots := make([]*cache.Snapshot, 0, len(views))
	releases := make([]func(), 0, len(views))

	for _, view := range views {
		snapshot, release, err := view.Snapshot()
		if err != nil {
			// Log but continue - skip views that can't produce snapshots
			continue
		}
		snapshots = append(snapshots, snapshot)
		releases = append(releases, release)
	}

	// Ensure all snapshots are released
	defer func() {
		for _, release := range releases {
			release()
		}
	}()

	if len(snapshots) == 0 {
		return nil, fmt.Errorf("no valid snapshots available")
	}

	// Use gopls's internal symbol search
	// Based on: gopls/internal/golang/workspace_symbol.go WorkspaceSymbols()
	symbols, err := golang.WorkspaceSymbols(
		ctx,
		settings.SymbolFuzzy,
		settings.DynamicSymbols,
		snapshots,
		params.Query,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search symbols: %w", err)
	}

	return symbols, nil
}

// All other protocol.Server methods return "not implemented" errors.
// These are stubs to satisfy the interface but are not used by core.

func (s *minimalServer) Progress(context.Context, *protocol.ProgressParams) error {
	return fmt.Errorf("not implemented")
}
func (s *minimalServer) SetTrace(context.Context, *protocol.SetTraceParams) error {
	return fmt.Errorf("not implemented")
}
func (s *minimalServer) IncomingCalls(context.Context, *protocol.CallHierarchyIncomingCallsParams) ([]protocol.CallHierarchyIncomingCall, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) OutgoingCalls(context.Context, *protocol.CallHierarchyOutgoingCallsParams) ([]protocol.CallHierarchyOutgoingCall, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) ResolveCodeAction(context.Context, *protocol.CodeAction) (*protocol.CodeAction, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) ResolveCodeLens(context.Context, *protocol.CodeLens) (*protocol.CodeLens, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) ResolveCommand(context.Context, *protocol.ExecuteCommandParams) (*protocol.ExecuteCommandParams, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) ResolveCompletionItem(context.Context, *protocol.CompletionItem) (*protocol.CompletionItem, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) ResolveDocumentLink(context.Context, *protocol.DocumentLink) (*protocol.DocumentLink, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) Exit(context.Context) error {
	return fmt.Errorf("not implemented")
}
func (s *minimalServer) Initialize(context.Context, *protocol.ParamInitialize) (*protocol.InitializeResult, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) Initialized(context.Context, *protocol.InitializedParams) error {
	return fmt.Errorf("not implemented")
}
func (s *minimalServer) Resolve(context.Context, *protocol.InlayHint) (*protocol.InlayHint, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) DidChangeNotebookDocument(context.Context, *protocol.DidChangeNotebookDocumentParams) error {
	return fmt.Errorf("not implemented")
}
func (s *minimalServer) DidCloseNotebookDocument(context.Context, *protocol.DidCloseNotebookDocumentParams) error {
	return fmt.Errorf("not implemented")
}
func (s *minimalServer) DidOpenNotebookDocument(context.Context, *protocol.DidOpenNotebookDocumentParams) error {
	return fmt.Errorf("not implemented")
}
func (s *minimalServer) DidSaveNotebookDocument(context.Context, *protocol.DidSaveNotebookDocumentParams) error {
	return fmt.Errorf("not implemented")
}
func (s *minimalServer) Shutdown(context.Context) error {
	return fmt.Errorf("not implemented")
}
func (s *minimalServer) CodeAction(context.Context, *protocol.CodeActionParams) ([]protocol.CodeAction, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) CodeLens(context.Context, *protocol.CodeLensParams) ([]protocol.CodeLens, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) ColorPresentation(context.Context, *protocol.ColorPresentationParams) ([]protocol.ColorPresentation, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) Completion(context.Context, *protocol.CompletionParams) (*protocol.CompletionList, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) Declaration(context.Context, *protocol.DeclarationParams) (*protocol.Or_textDocument_declaration, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) Definition(context.Context, *protocol.DefinitionParams) ([]protocol.Location, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) Diagnostic(context.Context, *protocol.DocumentDiagnosticParams) (*protocol.DocumentDiagnosticReport, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) DidChange(context.Context, *protocol.DidChangeTextDocumentParams) error {
	return fmt.Errorf("not implemented")
}
func (s *minimalServer) DidClose(context.Context, *protocol.DidCloseTextDocumentParams) error {
	return fmt.Errorf("not implemented")
}
func (s *minimalServer) DidOpen(context.Context, *protocol.DidOpenTextDocumentParams) error {
	return fmt.Errorf("not implemented")
}
func (s *minimalServer) DidSave(context.Context, *protocol.DidSaveTextDocumentParams) error {
	return fmt.Errorf("not implemented")
}

// DidChangeWatchedFiles notifies gopls that files have changed on disk.
// This is called by the file watcher when it detects filesystem changes.
// It implements the LSP protocol to ensure proper cache invalidation.
func (s *minimalServer) DidChangeWatchedFiles(ctx context.Context, params *protocol.DidChangeWatchedFilesParams) error {
	if len(params.Changes) == 0 {
		return nil
	}

	// Convert FileEvents to file.Modifications
	// This is the core internal API that gopls uses to process file changes
	modifications := s.fileEventsToModifications(params.Changes)

	// Call gopls's internal API directly
	// This handles all the cache invalidation, view updates, etc.
	_, err := s.session.DidModifyFiles(ctx, modifications)
	if err != nil {
		return fmt.Errorf("failed to process file changes: %w", err)
	}

	return nil
}

// fileEventsToModifications converts LSP FileEvents to file.Modifications
// This is the conversion that gopls uses internally when processing DidChangeWatchedFiles
func (s *minimalServer) fileEventsToModifications(events []protocol.FileEvent) []file.Modification {
	modifications := make([]file.Modification, 0, len(events))
	for _, event := range events {
		modifications = append(modifications, file.Modification{
			URI:    event.URI,
			Action: changeTypeToFileAction(event.Type),
			OnDisk: true, // Important: marks this as an on-disk change (not editor change)
		})
	}
	return modifications
}

// changeTypeToFileAction converts LSP FileChangeType to file.Action
// Based on gopls/internal/server/text_synchronization.go
func changeTypeToFileAction(ct protocol.FileChangeType) file.Action {
	switch ct {
	case protocol.Created:
		return file.Create
	case protocol.Changed:
		return file.Change
	case protocol.Deleted:
		return file.Delete
	default:
		return file.Change
	}
}
func (s *minimalServer) DocumentColor(context.Context, *protocol.DocumentColorParams) ([]protocol.ColorInformation, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) DocumentHighlight(context.Context, *protocol.DocumentHighlightParams) ([]protocol.DocumentHighlight, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) DocumentLink(context.Context, *protocol.DocumentLinkParams) ([]protocol.DocumentLink, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) DocumentSymbol(context.Context, *protocol.DocumentSymbolParams) ([]any, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) FoldingRange(context.Context, *protocol.FoldingRangeParams) ([]protocol.FoldingRange, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) Formatting(context.Context, *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) Hover(context.Context, *protocol.HoverParams) (*protocol.Hover, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) Implementation(context.Context, *protocol.ImplementationParams) ([]protocol.Location, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) InlayHint(context.Context, *protocol.InlayHintParams) ([]protocol.InlayHint, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) InlineCompletion(context.Context, *protocol.InlineCompletionParams) (*protocol.Or_Result_textDocument_inlineCompletion, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) InlineValue(context.Context, *protocol.InlineValueParams) ([]protocol.InlineValue, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) LinkedEditingRange(context.Context, *protocol.LinkedEditingRangeParams) (*protocol.LinkedEditingRanges, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) Moniker(context.Context, *protocol.MonikerParams) ([]protocol.Moniker, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) OnTypeFormatting(context.Context, *protocol.DocumentOnTypeFormattingParams) ([]protocol.TextEdit, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) PrepareCallHierarchy(context.Context, *protocol.CallHierarchyPrepareParams) ([]protocol.CallHierarchyItem, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) PrepareRename(context.Context, *protocol.PrepareRenameParams) (*protocol.PrepareRenameResult, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) PrepareTypeHierarchy(context.Context, *protocol.TypeHierarchyPrepareParams) ([]protocol.TypeHierarchyItem, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) RangeFormatting(context.Context, *protocol.DocumentRangeFormattingParams) ([]protocol.TextEdit, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) RangesFormatting(context.Context, *protocol.DocumentRangesFormattingParams) ([]protocol.TextEdit, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) References(context.Context, *protocol.ReferenceParams) ([]protocol.Location, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) Rename(context.Context, *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) SelectionRange(context.Context, *protocol.SelectionRangeParams) ([]protocol.SelectionRange, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) SemanticTokensFull(context.Context, *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) SemanticTokensFullDelta(context.Context, *protocol.SemanticTokensDeltaParams) (any, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) SemanticTokensRange(context.Context, *protocol.SemanticTokensRangeParams) (*protocol.SemanticTokens, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) SignatureHelp(context.Context, *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) TypeDefinition(context.Context, *protocol.TypeDefinitionParams) ([]protocol.Location, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) WillSave(context.Context, *protocol.WillSaveTextDocumentParams) error {
	return fmt.Errorf("not implemented")
}
func (s *minimalServer) WillSaveWaitUntil(context.Context, *protocol.WillSaveTextDocumentParams) ([]protocol.TextEdit, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) Subtypes(context.Context, *protocol.TypeHierarchySubtypesParams) ([]protocol.TypeHierarchyItem, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) Supertypes(context.Context, *protocol.TypeHierarchySupertypesParams) ([]protocol.TypeHierarchyItem, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) WorkDoneProgressCancel(context.Context, *protocol.WorkDoneProgressCancelParams) error {
	return fmt.Errorf("not implemented")
}
func (s *minimalServer) DiagnosticWorkspace(context.Context, *protocol.WorkspaceDiagnosticParams) (*protocol.WorkspaceDiagnosticReport, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) DidChangeConfiguration(context.Context, *protocol.DidChangeConfigurationParams) error {
	return fmt.Errorf("not implemented")
}
func (s *minimalServer) DidChangeWorkspaceFolders(context.Context, *protocol.DidChangeWorkspaceFoldersParams) error {
	return fmt.Errorf("not implemented")
}
func (s *minimalServer) DidCreateFiles(context.Context, *protocol.CreateFilesParams) error {
	return fmt.Errorf("not implemented")
}
func (s *minimalServer) DidDeleteFiles(context.Context, *protocol.DeleteFilesParams) error {
	return fmt.Errorf("not implemented")
}
func (s *minimalServer) DidRenameFiles(context.Context, *protocol.RenameFilesParams) error {
	return fmt.Errorf("not implemented")
}
func (s *minimalServer) ExecuteCommand(context.Context, *protocol.ExecuteCommandParams) (any, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) TextDocumentContent(context.Context, *protocol.TextDocumentContentParams) (*protocol.TextDocumentContentResult, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) WillCreateFiles(context.Context, *protocol.CreateFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) WillDeleteFiles(context.Context, *protocol.DeleteFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) WillRenameFiles(context.Context, *protocol.RenameFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *minimalServer) ResolveWorkspaceSymbol(context.Context, *protocol.WorkspaceSymbol) (*protocol.WorkspaceSymbol, error) {
	return nil, fmt.Errorf("not implemented")
}
