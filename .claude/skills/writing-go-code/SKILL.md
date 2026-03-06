---
name: writing-go-code
description: Apply Go coding standards when writing or modifying Go code. Use when implementing functions, using dependency injection, handling errors idiomatically, or working with interfaces. For test conventions, use the `writing-go-tests` skill instead.
---

# Go Development Standards

Project-specific Go coding standards for this codebase.

## Companion Skills

- **`applying-effective-go`** — General Go idioms from the official Effective Go documentation (naming, control flow, error handling philosophy, concurrency patterns). Complementary to this skill.
- **`writing-go-tests`** — Test conventions, mock usage, assertions, naming. Always load when writing test files.

## Code Organization

```go
// 1. Struct definition
type MyService struct {
    logger Logger
    fs     FileSystem
}

// 2. Interface verification (immediately after struct)
var _ Service = (*MyService)(nil)

// 3. Constructor with dependency injection
func NewMyService(logger Logger, fs FileSystem) *MyService {
    return &MyService{logger: logger, fs: fs}
}
```

## Dependency Injection

Always inject dependencies via constructors. Never create dependencies internally.

```go
// Good: dependencies injected
func NewHandler(
    logger Logger,
    service Service,
    validator Validator,
) *Handler {
    return &Handler{
        logger:    logger,
        service:   service,
        validator: validator,
    }
}

// Bad: dependencies created internally
func NewHandler() *Handler {
    return &Handler{
        logger:    NewDefaultLogger(),  // Don't do this
        service:   NewService(),        // Don't do this
    }
}
```

## Testability

Every external dependency (OS calls, file I/O, command execution, environment access, network, time) must be behind an interface so unit tests can inject mocks. This is non-negotiable — code that calls `os.*`, `exec.*`, or similar directly in business logic is untestable.

**The pattern**:

1. Define an interface describing the capability
2. Create a `Default*` concrete implementation that wraps the real calls
3. Constructor returns the concrete type (not the interface)
4. Business logic accepts the interface, never the concrete type
5. The composition root (e.g., `cmd/add.go`) wires concrete types to interfaces

```go
// 1. Interface — what consumers depend on
type FileSystem interface {
    ReadFileContents(path string) ([]byte, error)
    PathExists(path string) (bool, error)
}

// 2. Concrete implementation — wraps real OS calls
type DefaultFileSystem struct{}

func NewDefaultFileSystem() *DefaultFileSystem {
    return &DefaultFileSystem{}
}

func (fs *DefaultFileSystem) ReadFileContents(path string) ([]byte, error) {
    return os.ReadFile(path)
}

// 3. Business logic — accepts interface, never calls os.* directly
func LoadConfig(fs FileSystem, path string) (Config, error) {
    data, err := fs.ReadFileContents(path)
    // ...
}

// 4. Composition root — wires concrete to interface
func runAdd(cmd *cobra.Command, args []string) error {
    fs := utils.NewDefaultFileSystem()
    cfg, err := config.LoadConfig(fs, configPath)
    // ...
}
```

**What must be behind an interface**:

- File I/O (`os.ReadFile`, `os.WriteFile`, `os.MkdirAll`, etc.) → `FileSystem`
- Command execution (`os/exec`) → `Commander`
- Environment variables (`os.Getenv`) → `EnvironmentManager`
- Time, network, or any other non-deterministic dependency

**What does NOT need an interface**:

- Pure functions (string manipulation, data transformation)
- Standard library types used as values (`time.Duration`, `filepath.Join`)
- CLI framework wiring (Cobra commands, flag parsing)

## Mock Generation

Mocks use `mockery` with moq template. To regenerate all mocks:

```bash
mockery
```

Mock types are prefixed with `Moq` (e.g., `MoqLogger`, `MoqFileSystem`). For mock usage conventions in tests, see the `writing-go-tests` skill.

## Optional Types

Use `samber/mo` for safer nil handling:

```go
import "github.com/samber/mo"

type Config struct {
    Shell mo.Option[string]
}

if shell, ok := config.Shell.Get(); ok {
    // use shell
}
```

## Code Formatting

- Line length: 120 characters max.
- Vertically align function arguments when there are multiple arguments.
- Insert blank lines between logical sections of code.
- Do not separate error unwrapping from related code with a blank line; treat it as part of the same section.

```go
// Good: error handling is part of the same section
result, err := doSomething()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// Next logical section starts after blank line
processResult(result)
```

## Documentation

End all type and function comments with a period, following Go conventions.

```go
// MyService handles business logic for the application.
type MyService struct {
    // ...
}

// Process executes the main workflow and returns the result.
func (s *MyService) Process(ctx context.Context) error {
    // ...
}
```

## Key Rules

- Use the Go standard library whenever possible. Only use third-party libraries when necessary.
- Pre-allocate slices/maps when size is known.
- All external dependencies must be behind interfaces — see the Testability section above for the full pattern.
- All non-CLI codepaths must be unit-testable via mock injection. If a function can't be tested without hitting the real OS, it needs refactoring.
- Never edit mock files manually.
