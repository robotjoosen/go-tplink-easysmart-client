# AGENTS.md

## Build & Test Commands

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run a single test
go test -v -run Test_Client ./...

# Run tests with race detection
go test -race ./...

# Build the project
go build ./...

# Run go vet
go vet ./...

# Format code
go fmt ./...
```

## Project Structure

```
.
├── client.go              # Main client interface and factory
├── client_test.go         # Integration tests
├── docs/
│   └── TLPP.md            # TLPP protocol specification
├── internal/
│   ├── gui/               # GUI/HTTP implementation
│   │   ├── client.go
│   │   └── parse.go       # HTML parsing utilities
│   └── tlpp/              # TLPP/UDP implementation
│       └── client.go
└── pkg/
    ├── errors/             # Custom error definitions
    │   └── errors.go
    └── model/             # Data models with JSON tags
        └── model.go
```

## Code Style

### Imports
Group imports in order: stdlib, then third-party, then internal packages.
```go
import (
    "context"
    "errors"
    "net/http"

    clientErrors "github.com/robotjoosen/go-tplink-easysmart-client/pkg/errors"
    "github.com/robotjoosen/go-tplink-easysmart-client/pkg/model"
)
```

### Error Handling
- Use `errors.Join()` to wrap errors with context
- Define sentinel errors in `pkg/errors/errors.go`
- Return wrapped errors with descriptive context

```go
// pkg/errors/errors.go
var (
    ErrAuthenticationFailed = errors.New("authentication failed")
    ErrInvalidRequest       = errors.New("invalid request")
    ErrInvalidResponse      = errors.New("invalid response")
)

// Usage
return errors.Join(err, clientErrors.ErrInvalidRequest)
```

### Context
- All public methods must accept `context.Context` as first parameter
- Propagate context through the call chain
- Use `http.NewRequestWithContext` for HTTP requests

### Naming Conventions
- **Interfaces**: `ClientAware` (suffix with -er/-able)
- **Structs**: `Client`, `Device`, `Port`
- **Constants**: `PathLogin`, `tlppTXPort` (mixed case for exported, camelCase for unexported)
- **Packages**: single lowercase word or domain path

### Types & Structs
- Use struct tags for JSON serialization
- Export fields that need serialization
- Use meaningful type names (e.g., `mode string` for mode enums)

```go
type Device struct {
    Description     string `json:"description"`
    MAC             string `json:"mac"`
}

type mode string

const (
    GUI  mode = "gui"
    TLPP mode = "tlpp"
)
```

### Testing
- Test files: `*_test.go` in same package or `_test` package for integration tests
- Use `testify/assert` for assertions
- Use table-driven tests for multiple test cases

```go
func Test_Client(t *testing.T) {
    c := client.NewClient("192.168.2.101", client.TLPP)
    assert.NoError(t, c.Login(context.Background(), "admin", "password"))
}
```

## Patterns

### Client Factory Pattern
Use a factory function with mode enum to select implementation:
```go
func NewClient(address string, m mode) ClientAware {
    switch m {
    case GUI:
        return gui.NewClient(address)
    case TLPP:
        return tlpp.NewClient(address)
    default:
        return nil
    }
}
```

### Interface Definition
Define interfaces at the consumer level:
```go
type ClientAware interface {
    Login(ctx context.Context, username, password string) error
    GetDevice(ctx context.Context) (model.Device, error)
    GetPortStats(ctx context.Context) ([]model.Port, error)
    SetLED(ctx context.Context, state bool) error
}
```
