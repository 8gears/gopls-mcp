// Package api provides diagnostics and metadata types for all gopls MCP tools.
package api

import "time"

// WorkspaceDiagnostics provides metadata about tool execution and cache state.
// This helps users understand why a tool may have returned empty results or
// provides actionable suggestions for improving results.
type WorkspaceDiagnostics struct {
	// CacheStatus indicates the state of gopls's symbol cache.
	// Possible values: "empty", "partial", "full", "unknown"
	CacheStatus string `json:"cache_status,omitempty" jsonschema:"cache status (empty/partial/full/unknown)"`

	// IndexedPackages is the number of packages currently indexed in gopls's cache.
	IndexedPackages int `json:"indexed_packages,omitempty" jsonschema:"number of indexed packages"`

	// TotalPackages is the total number of packages in the workspace.
	TotalPackages int `json:"total_packages,omitempty" jsonschema:"total workspace packages"`

	// AnalysisDuration is how long the operation took.
	AnalysisDuration time.Duration `json:"analysis_duration,omitempty" jsonschema:"operation duration in milliseconds"`

	// Suggestions provide actionable advice to the user.
	// Examples: "Try building the project first", "Use analyze_workspace for discovery"
	Suggestions []string `json:"suggestions,omitempty" jsonschema:"actionable suggestions"`

	// FallbackUsed indicates whether a fallback method (e.g., grep) was used.
	FallbackUsed bool `json:"fallback_used,omitempty" jsonschema:"whether fallback search was used"`

	// FallbackReason explains why fallback was used.
	FallbackReason string `json:"fallback_reason,omitempty" jsonschema:"reason for using fallback"`
}

// SearchResultDiagnostics provides specific diagnostics for search operations.
type SearchResultDiagnostics struct {
	WorkspaceDiagnostics

	// QueryUsed is the search query that was executed.
	QueryUsed string `json:"query_used,omitempty" jsonschema:"the search query"`

	// SymbolsFound is the number of symbols found.
	SymbolsFound int `json:"symbols_found,omitempty" jsonschema:"number of symbols found"`

	// SearchMethod indicates which search method was used.
	// Values: "lsp", "grep", "hybrid"
	SearchMethod string `json:"search_method,omitempty" jsonschema:"search method used"`
}

// ReferenceDiagnostics provides specific diagnostics for symbol references.
type ReferenceDiagnostics struct {
	WorkspaceDiagnostics

	// SymbolSearched is the symbol that was searched for.
	SymbolSearched string `json:"symbol_searched,omitempty" jsonschema:"the symbol searched"`

	// ReferencesFound is the number of references found.
	ReferencesFound int `json:"references_found,omitempty" jsonschema:"number of references found"`

	// FilesSearched is the number of files searched.
	FilesSearched int `json:"files_searched,omitempty" jsonschema:"number of files searched"`
}

// AnalysisDiagnostics provides specific diagnostics for workspace analysis.
type AnalysisDiagnostics struct {
	WorkspaceDiagnostics

	// ProjectType is the detected project type (e.g., "module", "workspace", "gopath").
	ProjectType string `json:"project_type,omitempty" jsonschema:"detected project type"`

	// ModulesFound is the number of modules discovered.
	ModulesFound int `json:"modules_found,omitempty" jsonschema:"number of modules found"`

	// EntryPointsFound is the number of entry points discovered.
	EntryPointsFound int `json:"entry_points_found,omitempty" jsonschema:"number of entry points found"`
}

// GetDefaultSuggestions returns common suggestions based on diagnostics state.
func (d *WorkspaceDiagnostics) GetDefaultSuggestions() []string {
	if d.Suggestions == nil {
		d.Suggestions = []string{}
	}

	// Add suggestions based on cache status
	switch d.CacheStatus {
	case "empty":
		d.Suggestions = append(d.Suggestions, "Cache is empty - try building the project first with 'go build ./...'")
	case "partial":
		if d.IndexedPackages < d.TotalPackages {
			d.Suggestions = append(d.Suggestions, "Only some packages are indexed - use 'analyze_workspace' to discover all packages")
		}
	}

	// Add suggestions if fallback was used
	if d.FallbackUsed && d.FallbackReason != "" {
		d.Suggestions = append(d.Suggestions, "Used fallback search: "+d.FallbackReason)
	}

	return d.Suggestions
}

// SetPartialCache sets the cache status to partial with package counts.
func (d *WorkspaceDiagnostics) SetPartialCache(indexed, total int) {
	d.CacheStatus = "partial"
	d.IndexedPackages = indexed
	d.TotalPackages = total
}

// SetFullCache sets the cache status to full.
func (d *WorkspaceDiagnostics) SetFullCache(total int) {
	d.CacheStatus = "full"
	d.IndexedPackages = total
	d.TotalPackages = total
}

// MarkFallbackUsed marks that a fallback method was used.
func (d *WorkspaceDiagnostics) MarkFallbackUsed(reason string) {
	d.FallbackUsed = true
	d.FallbackReason = reason
}
