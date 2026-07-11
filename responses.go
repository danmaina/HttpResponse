package handlers

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/danmaina/logger/v2"
)

// ResponseEncoder defines the interface for encoding response payloads.
type ResponseEncoder interface {
	Encode(w http.ResponseWriter, status int, err error, body interface{}) error
	ContentType() string
}

// PayloadFormatter defines the interface a payload can implement to configure the serialization format.
type PayloadFormatter interface {
	ResponseFormat() string // returns the format name, e.g., "json", "xml", "soap"
}

// APIHandler is a custom handler type that returns status code, error, and body payload.
type APIHandler func(w http.ResponseWriter, r *http.Request) (status int, err error, body interface{})

// Handler wraps an APIHandler to conform to the standard http.HandlerFunc.
func Handler(h APIHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err, body := h(w, r)
		ReturnResponse(w, r, status, err, body, "")
	}
}

// HandlerWithFormat wraps an APIHandler using a specific response format.
func HandlerWithFormat(h APIHandler, format string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err, body := h(w, r)
		ReturnResponse(w, r, status, err, body, format)
	}
}

// Response represents the standard response payload envelope.
type Response struct {
	XMLName xml.Name      `json:"-" xml:"response"`
	Status  int           `json:"status" xml:"status"`
	Error   error         `json:"error" xml:"error"`
	Body    interface{}   `json:"body" xml:"body"`
	Format  string        `json:"-" xml:"-"`
	Request *http.Request `json:"-" xml:"-"`
}

var (
	encoders = make(map[string]ResponseEncoder)
	mu       sync.RWMutex
)

// RegisterEncoder registers a custom encoder for a specific format.
func RegisterEncoder(format string, encoder ResponseEncoder) {
	mu.Lock()
	defer mu.Unlock()
	encoders[strings.ToLower(format)] = encoder
}

// GetEncoder retrieves the registered encoder for the given format.
func GetEncoder(format string) (ResponseEncoder, bool) {
	mu.RLock()
	defer mu.RUnlock()
	enc, ok := encoders[strings.ToLower(format)]
	return enc, ok
}

func init() {
	RegisterEncoder("json", JSONEncoder{})
	RegisterEncoder("xml", XMLEncoder{})
}

// ReturnResponse is the main entry point to package and return HTTP responses.
// Arguments are ordered following the standard net/http handler signature.
// format can be "json", "xml", or "both". If empty, it defaults to "both" (content negotiation).
func ReturnResponse(w http.ResponseWriter, r *http.Request, status int, err error, body interface{}, format string) {
	_ = Response{
		Status:  status,
		Error:   err,
		Body:    body,
		Format:  format,
		Request: r,
	}.Send(w)
}

// jsonResponseRepresentation is used internally to avoid encoding errors caused by Go's error interface.
type jsonResponseRepresentation struct {
	Status int         `json:"status"`
	Error  *string     `json:"error,omitempty"`
	Body   interface{} `json:"body,omitempty"`
}

// xmlResponseRepresentation is used internally to structure the XML output.
type xmlResponseRepresentation struct {
	XMLName xml.Name    `xml:"response"`
	Status  int         `xml:"status"`
	Error   *string     `xml:"error,omitempty"`
	Body    interface{} `xml:"body,omitempty"`
}

// Send serializes and writes the response payload to the response writer using the configured encoder.
func (res Response) Send(w http.ResponseWriter) error {
	// 1. Determine format configuration
	format := res.Format

	// Check if the body implements PayloadFormatter
	if format == "" {
		if pf, ok := res.Body.(PayloadFormatter); ok {
			format = pf.ResponseFormat()
		}
	}

	// Default to "both" if still empty
	if format == "" {
		format = "both"
	}

	// 2. Perform Content Negotiation if format is "both"
	if format == "both" {
		if res.Request != nil {
			accept := res.Request.Header.Get("Accept")
			if strings.Contains(accept, "application/xml") || strings.Contains(accept, "text/xml") {
				format = "xml"
			} else {
				format = "json"
			}
		} else {
			// Fall back to json if no request is provided for negotiation
			format = "json"
		}
	}

	// 3. Write Response using the selected encoder
	enc, ok := GetEncoder(format)
	if !ok {
		logger.WARN("Unknown response format: ", format, ", falling back to JSON")
		enc, _ = GetEncoder("json")
	}

	return enc.Encode(w, res.Status, res.Error, res.Body)
}

// JSONEncoder implements ResponseEncoder for JSON serialization.
type JSONEncoder struct{}

func (JSONEncoder) ContentType() string {
	return "application/json"
}

func (JSONEncoder) Encode(w http.ResponseWriter, status int, err error, body interface{}) error {
	logger.INFO("Setting the Response Header to json")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err != nil {
		logger.ERR("Returning Error Body (JSON): ", err)
		return json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
	}

	logger.DEBUG("Creating a new JSON Encoder")
	if errE := json.NewEncoder(w).Encode(body); errE != nil {
		logger.ERR("Error while encoding the Response Body: ", errE)
		errStr := errE.Error()
		return json.NewEncoder(w).Encode(jsonResponseRepresentation{
			Status: status,
			Error:  &errStr,
			Body:   body,
		})
	}
	return nil
}

// XMLEncoder implements ResponseEncoder for XML serialization.
type XMLEncoder struct{}

func (XMLEncoder) ContentType() string {
	return "application/xml"
}

func (XMLEncoder) Encode(w http.ResponseWriter, status int, err error, body interface{}) error {
	logger.INFO("Setting the Response Header to xml")
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)

	if err != nil {
		logger.ERR("Returning Error Body (XML): ", err)
		errStr := err.Error()
		return xml.NewEncoder(w).Encode(xmlResponseRepresentation{
			Status: status,
			Error:  &errStr,
		})
	}

	logger.DEBUG("Creating a new XML Encoder")
	b, errE := marshalXML(body)
	if errE != nil {
		logger.ERR("Error while encoding the Response Body to XML: ", errE)
		errStr := errE.Error()
		return xml.NewEncoder(w).Encode(xmlResponseRepresentation{
			Status: status,
			Error:  &errStr,
			Body:   body,
		})
	}

	output := string(b)
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		_, errW := w.Write([]byte("<response></response>"))
		return errW
	}

	if !strings.HasPrefix(trimmed, "<") {
		output = "<response>" + output + "</response>"
	}

	_, errW := w.Write([]byte(output))
	return errW
}

// marshalXML is a helper to recursively marshal any interface (including maps and slices) to XML.
func marshalXML(v interface{}) ([]byte, error) {
	if v == nil {
		return nil, nil
	}

	val := reflect.ValueOf(v)
	kind := val.Kind()

	// Handle pointers
	if kind == reflect.Ptr {
		if val.IsNil() {
			return nil, nil
		}
		return marshalXML(val.Elem().Interface())
	}

	// Handle structs (use standard xml.Marshal)
	if kind == reflect.Struct {
		return xml.Marshal(v)
	}

	// Handle maps
	if kind == reflect.Map {
		var buf bytes.Buffer
		buf.WriteString("<response>")
		for _, key := range val.MapKeys() {
			kStr := fmt.Sprintf("%v", key.Interface())
			vVal := val.MapIndex(key).Interface()
			vBytes, err := marshalXML(vVal)
			if err != nil {
				return nil, err
			}
			buf.WriteString(fmt.Sprintf("<%s>", kStr))
			buf.Write(vBytes)
			buf.WriteString(fmt.Sprintf("</%s>", kStr))
		}
		buf.WriteString("</response>")
		return buf.Bytes(), nil
	}

	// Handle slices and arrays
	if kind == reflect.Slice || kind == reflect.Array {
		// Special case: []byte
		if val.Type().Elem().Kind() == reflect.Uint8 {
			var escBuf bytes.Buffer
			if err := xml.EscapeText(&escBuf, val.Bytes()); err != nil {
				return nil, err
			}
			return escBuf.Bytes(), nil
		}

		var buf bytes.Buffer
		buf.WriteString("<elements>")
		for i := 0; i < val.Len(); i++ {
			elemBytes, err := marshalXML(val.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			buf.WriteString("<element>")
			buf.Write(elemBytes)
			buf.WriteString("</element>")
		}
		buf.WriteString("</elements>")
		return buf.Bytes(), nil
	}

	// Escape primitive values for XML text context
	var escBuf bytes.Buffer
	if err := xml.EscapeText(&escBuf, []byte(fmt.Sprintf("%v", v))); err != nil {
		return nil, err
	}
	return escBuf.Bytes(), nil
}

// loggingResponseWriter captures the response body for logging.
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	lrw.body.Write(b)
	return lrw.ResponseWriter.Write(b)
}

// generateTitle converts an endpoint like /api/v1/organizations/countries into "Organizations Countries"
func generateTitle(method, urlPath string) string {
	segments := strings.Split(urlPath, "/")
	var words []string
	
	for _, s := range segments {
		if s == "" || s == "api" || s == "v1" || len(s) > 30 {
			continue
		}
		// Skip if it's a number
		if _, err := strconv.Atoi(s); err == nil {
			continue
		}
		words = append(words, strings.Title(strings.ToLower(s)))
	}
	
	methodTitle := strings.Title(strings.ToLower(method))
	return fmt.Sprintf("%s %s", methodTitle, strings.Join(words, " "))
}

// PayloadLoggingMiddleware logs incoming request and outgoing response payloads.
func PayloadLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		reqBody := []byte{}
		if r.Body != nil {
			reqBody, _ = io.ReadAll(r.Body)
			// Restore the io.ReadCloser to its original state
			r.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}

		title := generateTitle(r.Method, r.URL.Path)

		reqLogData := map[string]interface{}{
			"headers": r.Header,
			"payload": string(reqBody),
		}
		reqJSON, _ := json.MarshalIndent(reqLogData, "", "  ")
		logger.INFO(fmt.Sprintf("\n%s Request:\n\n%s\n", title, string(reqJSON)))

		lrw := &loggingResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			body:           bytes.NewBuffer(nil),
		}

		next.ServeHTTP(lrw, r)

		duration := time.Since(startTime)
		
		resLogData := map[string]interface{}{
			"statusCode": lrw.statusCode,
			"duration":   duration.String(),
			"payload":    lrw.body.String(),
		}
		resJSON, _ := json.MarshalIndent(resLogData, "", "  ")
		logger.INFO(fmt.Sprintf("\n%s Response:\n\n%s\n", title, string(resJSON)))
	})
}

