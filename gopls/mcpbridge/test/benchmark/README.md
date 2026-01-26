# gopls-mcp Benchmark Suite

This directory contains the performance benchmark suite for `gopls-mcp`. It measures the execution time of MCP tools and compares them against traditional Go CLI tools (`go list`, `grep`, `go build`, etc.).

## Usage

### Prerequisites

- Go 1.25 or later
- A Go project to benchmark against (defaults to the `gopls` codebase itself)

### Running Benchmarks

1. **Run the suite:**

   ```bash
   go run benchmark_main.go -compare
   ```

   This will execute the benchmarks and generate a `benchmark_results.json` file.

2. **Generate the Report:**

   ```bash
   go run reportgen/main.go benchmark_results.json > RESULTS.md
   ```

   This converts the JSON data into a human-readable Markdown report.

### Options

| Flag | Description | Default |
|------|-------------|---------|
| `-compare` | Run comparison benchmarks against CLI tools | `true` |
| `-output` | Output file for JSON results | `benchmark_results.json` |
| `-project` | Absolute path to the project to benchmark | `../../..` (gopls-mcp root) |
| `-verbose` | Enable verbose logging | `false` |

## Output Files

*   **`benchmark_results.json`**: Raw data from the run.
*   **`RESULTS.md`**: Generated report. **Note:** This file is ignored by git (`.gitignore`) to prevent local runs from overwriting the official baseline.
*   **`BASELINE_RESULTS.md`**: A reference report from a standard environment, checked into the repository for comparison.

## Methodology

The benchmarks use **adaptive sampling** to ensure statistical significance.
- **Warmup**: 3 iterations (discarded).
- **Sampling**: Runs until the Coefficient of Variation (CV) is low or max iterations are reached.
- **Comparisons**: Calculated as `Mean(CLI) / Mean(MCP)`.

For full technical details, see [METHODOLOGY.md](METHODOLOGY.md).

## Contributing

To add a new benchmark:
1.  Add the logic in `internal/benchmarks.go`.
2.  Register it in `benchmark_main.go`.
3.  If possible, add a traditional CLI comparison for speedup calculation.