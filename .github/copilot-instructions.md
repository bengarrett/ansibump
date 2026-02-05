# ANSIbump Copilot Instructions

## Project Overview

ANSIbump is a Go package that converts ANSI escape sequences (colors, cursor movements, character deletions) into HTML output. It's a library used primarily to render retro ANSI art and text files as interactive web content, particularly for the Defacto2 archive.

The core architecture consists of:
- **Decoder**: Main state machine that processes ANSI escape sequences byte-by-byte
- **Customizer**: Configuration wrapper for creating decoders with custom palettes, character sets, and parsing modes
- **Palette/Colors**: ANSI color mapping system supporting multiple color schemes (CGA, Xterm, Amiga DPaint2)
- **HTML Output**: Uses Go's `html/template` to generate safe HTML with inline styles

## Build, Test, and Lint Commands

### Using Task (Recommended)
This project uses [Task](https://taskfile.dev/) as the task runner. Common commands:

```bash
task test          # Run full test suite
task testr         # Run tests with race detection (slower but safer)
task lint          # Run formatter and linters
task nil           # Run nilaway static analysis for nil dereferences
task pkg-patch     # Update patch-level dependencies
task pkg-update    # Update all dependencies
task doc           # Generate and browse Go documentation locally
```

### Direct Go Commands
```bash
go test -count 1 ./...           # Run tests (count=1 avoids caching)
go test -count 1 -race ./...     # Run tests with race detection
gofumpt -l -w .                  # Format code
golangci-lint run -c .golangci.yaml  # Run all linters
go tool nilaway -test=false ./...    # Check for nil dereferences
```

## Key Architecture Patterns

### ANSI Escape Sequence Parsing
- Sequences start with ESC (0x1b) followed by CSI commands (Control Sequence Introducer: `[`)
- Parameters are semicolon-separated integers
- Parser handles:
  - **4-bit colors**: Standard ANSI codes (30-37 foreground, 40-47 background)
  - **8-bit colors**: Extended 256-color mode (ESC[38;5;Nm)
  - **24-bit RGB**: True color mode (ESC[38;2;R;G;Bm)
  - **Cursor control**: Up, down, forward, back, positioning, line operations
  - **Text attributes**: Bold, underline, invert, reset

### Color Model
- `Palette` enum defines color schemes: `CGA16`, `Xterm16`, `DP2` (Amiga)
- `Colors` array maps ANSI color indices (0-15) to hex values
- `Color` is a string type representing hex RGB values (e.g., "a50")
- Color conversion functions: `XtermHex()`, `XtermColors()`, `RGBHex()`, `Bright()`

### State Machine Decoder
The `Decoder` type maintains:
- Current cell position (row, column, width)
- Active text attributes (bold, underline, invert, fg/bg colors)
- Cursor position stack for save/restore
- Character grid for handling cursor movements and erasures
- Encoding support via `golang.org/x/text/encoding/charmap`

### HTML Generation
- Output wraps content in `<div>` with inline styles
- Each styled text segment gets its own `<span>` with `style="color:#...;background-color:#...;"`
- Uses `html.EscapeString()` to prevent injection
- Supports multiple character encodings: CodePage437, ISO-8859-1, etc.

## Code Conventions

### Error Handling
- Predefined `var` errors at package level (e.g., `ErrReader`, `ErrUnrecognized`)
- Functions return `error` as second return value
- Some functions return `(int64, error)` for byte counts written

### Constants
- ANSI control codes defined as named constants (e.g., `Bold = 1`, `ESC = 0x1b`)
- Color index ranges: `FG1st=30, FGEnd=37` for standard colors; `BrightFG1st=90, BrightFGEnd=97` for bright
- Special indices: `SetFG=38, SetBG=48` for 256/RGB color modes

### API Design
- Three main entry points for different output types:
  - `WriteTo(reader, writer, width)` - Stream-based with io.Writer
  - `Bytes(reader, width)` - Returns `[]byte`
  - `String(reader, width)` - Returns `string`
- `Customizer` type allows configuring color palette, character set, and parsing behavior
- Exported functions are capitalized; internal helpers are lowercase (Go convention)

### Testing
- Example tests in `*_test.go` files demonstrate public API usage
- Uses `github.com/nalgeon/be` assertion library (lightweight assertions)
- Tests use `t.Parallel()` for concurrent execution
- Fixtures use raw ANSI strings with escape sequences (e.g., `"\x1b[0m\x1b[5;33;42mHI\x1b[0m"`)

### Performance Notes
- `pipeReplaceAll()` is a custom replacement to avoid intermediate buffers
- Complex functions marked with linter directives: `//nolint:gocyclo,gocognit` for high complexity
- `bufio.Scanner` used for line-by-line processing to manage memory

## Linter Configuration

Configured in `.golangci.yaml`:
- Disabled linters: `depguard`, `exhaustruct`, `funlen`, `nlreturn`, `varnamelen`, `wsl`
- Formatters: `gofumpt`, `goimports`, `gci`, `gofmt`
- Test files exclude: `lll` (long line length)
- Temporary disabled: `cyclop`, `godot`, `godox`

## Known Patterns in Tests

- Examples use `fmt.Printf()` for output validation
- `be.Equal()` for assertions
- Test fixtures encode ANSI sequences as hex escapes for clarity
- Tests cover edge cases like out-of-range color indices and empty parameters

## Dependencies

- `golang.org/x/text/encoding/charmap` - Character encoding support
- `github.com/nalgeon/be` - Assertion library (test only)
- `go.uber.org/nilaway` - Nil dereference detection tool
