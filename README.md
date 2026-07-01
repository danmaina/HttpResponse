# HttpResponse (v2)

A clean, extensible Go package to structure, negotiate, and return HTTP responses (JSON, XML, SOAP, etc.).

## Features

- **Extensible Registry Pattern**: Add custom encoders (e.g. SOAP) via `RegisterEncoder`.
- **Automatic Content Negotiation**: Defaults to negotiating the format based on the request `Accept` header (supporting JSON, XML, and custom registered mime types).
- **Payload-level Formatting Config**: Payloads can implement `PayloadFormatter` to dynamically configure formatting.
- **APIHandler Wrapper**: Eliminate repetitive `return` boilerplate inside HTTP handlers.

## Usage

### 1. Extensible Formatter Registration (e.g. SOAP)
Pre-registered formats include `"json"` and `"xml"`. You can register custom formats in `init()`:

```go
import "github.com/danmaina/HttpResponse/v2"

type SOAPEncoder struct{}

func (SOAPEncoder) ContentType() string {
    return "application/soap+xml"
}

func (SOAPEncoder) Encode(w http.ResponseWriter, status int, err error, body interface{}) error {
    w.Header().Set("Content-Type", "application/soap+xml")
    w.WriteHeader(status)
    // custom serialization logic ...
    return nil
}

func init() {
    handlers.RegisterEncoder("soap", SOAPEncoder{})
}
```

### 2. Standard Usage
Invoke `ReturnResponse` in your handler:

```go
func MyHandler(w http.ResponseWriter, r *http.Request) {
    handlers.ReturnResponse(w, r, http.StatusOK, nil, myPayload, "json")
}
```

### 3. APIHandler Style (No explicit return statements needed)
To avoid manual returns, wrap your handler with `handlers.Handler`:

```go
func CreateUser(w http.ResponseWriter, r *http.Request) (status int, err error, body interface{}) {
    var user User
    if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
        return http.StatusBadRequest, err, nil // automatically returned as JSON/XML error
    }
    // business logic ...
    return http.StatusCreated, nil, user
}

// In router setup:
r.HandleFunc("/users", handlers.Handler(CreateUser)).Methods("POST")
```
