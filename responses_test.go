package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestJSONEncoder tests standard JSON encoding on success and error.
func TestJSONEncoder(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := map[string]string{"foo": "bar"}
		ReturnResponse(w, nil, http.StatusOK, nil, body, "json")

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", contentType)
		}
		expectedBody := `{"foo":"bar"}`
		actualBody := strings.TrimSpace(w.Body.String())
		if actualBody != expectedBody {
			t.Errorf("expected body %q, got %q", expectedBody, actualBody)
		}
	})

	t.Run("error", func(t *testing.T) {
		w := httptest.NewRecorder()
		err := errors.New("something went wrong")
		ReturnResponse(w, nil, http.StatusBadRequest, err, nil, "json")

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", contentType)
		}
		expectedBody := `{"error":"something went wrong"}`
		actualBody := strings.TrimSpace(w.Body.String())
		if actualBody != expectedBody {
			t.Errorf("expected body %q, got %q", expectedBody, actualBody)
		}
	})
}

// TestXMLEncoder tests standard XML encoding on success and error.
func TestXMLEncoder(t *testing.T) {
	t.Run("success struct", func(t *testing.T) {
		type TestStruct struct {
			Foo string `xml:"foo"`
		}
		w := httptest.NewRecorder()
		body := TestStruct{Foo: "bar"}
		ReturnResponse(w, nil, http.StatusOK, nil, body, "xml")

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if contentType := w.Header().Get("Content-Type"); contentType != "application/xml" {
			t.Errorf("expected Content-Type application/xml, got %q", contentType)
		}
		expectedBody := `<TestStruct><foo>bar</foo></TestStruct>`
		actualBody := strings.TrimSpace(w.Body.String())
		if actualBody != expectedBody {
			t.Errorf("expected body %q, got %q", expectedBody, actualBody)
		}
	})

	t.Run("success map", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := map[string]string{"foo": "bar"}
		ReturnResponse(w, nil, http.StatusOK, nil, body, "xml")

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		// Since maps are unordered in Go, we just check if it wraps in <response> and contains <foo>bar</foo>
		actualBody := strings.TrimSpace(w.Body.String())
		if !strings.HasPrefix(actualBody, "<response>") || !strings.HasSuffix(actualBody, "</response>") {
			t.Errorf("expected XML wrapped in response tag, got %q", actualBody)
		}
		if !strings.Contains(actualBody, "<foo>bar</foo>") {
			t.Errorf("expected XML to contain <foo>bar</foo>, got %q", actualBody)
		}
	})

	t.Run("success slice of primitives", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := []string{"foo", "bar"}
		ReturnResponse(w, nil, http.StatusOK, nil, body, "xml")

		actualBody := strings.TrimSpace(w.Body.String())
		expectedBody := `<elements><element>foo</element><element>bar</element></elements>`
		if actualBody != expectedBody {
			t.Errorf("expected body %q, got %q", expectedBody, actualBody)
		}
	})

	t.Run("success primitive value", func(t *testing.T) {
		w := httptest.NewRecorder()
		ReturnResponse(w, nil, http.StatusOK, nil, "simple string", "xml")

		actualBody := strings.TrimSpace(w.Body.String())
		expectedBody := `<response>simple string</response>`
		if actualBody != expectedBody {
			t.Errorf("expected body %q, got %q", expectedBody, actualBody)
		}
	})

	t.Run("error", func(t *testing.T) {
		w := httptest.NewRecorder()
		err := errors.New("xml error message")
		ReturnResponse(w, nil, http.StatusInternalServerError, err, nil, "xml")

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
		if contentType := w.Header().Get("Content-Type"); contentType != "application/xml" {
			t.Errorf("expected Content-Type application/xml, got %q", contentType)
		}
		expectedBody := `<response><status>500</status><error>xml error message</error></response>`
		actualBody := strings.TrimSpace(w.Body.String())
		if actualBody != expectedBody {
			t.Errorf("expected body %q, got %q", expectedBody, actualBody)
		}
	})
}

// TestContentNegotiation tests the content negotiation behavior when format is "both".
func TestContentNegotiation(t *testing.T) {
	type TestStruct struct {
		Val string `json:"val" xml:"val"`
	}
	body := TestStruct{Val: "negotiate"}

	t.Run("negotiate JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept", "application/json, text/plain, */*")
		ReturnResponse(w, req, http.StatusOK, nil, body, "both")

		if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", contentType)
		}
		expectedBody := `{"val":"negotiate"}`
		actualBody := strings.TrimSpace(w.Body.String())
		if actualBody != expectedBody {
			t.Errorf("expected body %q, got %q", expectedBody, actualBody)
		}
	})

	t.Run("negotiate XML", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept", "text/xml, application/xml, */*")
		ReturnResponse(w, req, http.StatusOK, nil, body, "both")

		if contentType := w.Header().Get("Content-Type"); contentType != "application/xml" {
			t.Errorf("expected Content-Type application/xml, got %q", contentType)
		}
		expectedBody := `<TestStruct><val>negotiate</val></TestStruct>`
		actualBody := strings.TrimSpace(w.Body.String())
		if actualBody != expectedBody {
			t.Errorf("expected body %q, got %q", expectedBody, actualBody)
		}
	})

	t.Run("negotiate default JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		ReturnResponse(w, req, http.StatusOK, nil, body, "both")

		if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", contentType)
		}
	})
}

type customPayload struct {
	format string
}

func (p customPayload) ResponseFormat() string {
	return p.format
}

// TestPayloadFormatter tests when the payload itself configures the format.
func TestPayloadFormatter(t *testing.T) {
	t.Run("payload requests XML", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := customPayload{format: "xml"}
		ReturnResponse(w, nil, http.StatusOK, nil, body, "")

		if contentType := w.Header().Get("Content-Type"); contentType != "application/xml" {
			t.Errorf("expected Content-Type application/xml, got %q", contentType)
		}
	})

	t.Run("payload requests JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := customPayload{format: "json"}
		ReturnResponse(w, nil, http.StatusOK, nil, body, "")

		if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", contentType)
		}
	})
}

// mockSOAPEncoder is a mock SOAP response encoder to test extensibility.
type mockSOAPEncoder struct{}

func (mockSOAPEncoder) ContentType() string {
	return "application/soap+xml"
}

func (mockSOAPEncoder) Encode(w http.ResponseWriter, status int, err error, body interface{}) error {
	w.Header().Set("Content-Type", "application/soap+xml")
	w.WriteHeader(status)
	if err != nil {
		_, errW := w.Write([]byte("<soap:Envelope><soap:Body><soap:Fault><faultstring>" + err.Error() + "</faultstring></soap:Fault></soap:Body></soap:Envelope>"))
		return errW
	}
	_, errW := w.Write([]byte("<soap:Envelope><soap:Body><response>soap success</response></soap:Body></soap:Envelope>"))
	return errW
}

// TestExtensibility tests registration and usage of custom encoders (like SOAP).
func TestExtensibility(t *testing.T) {
	RegisterEncoder("soap", mockSOAPEncoder{})

	t.Run("custom soap format success", func(t *testing.T) {
		w := httptest.NewRecorder()
		ReturnResponse(w, nil, http.StatusOK, nil, "body data", "soap")

		if contentType := w.Header().Get("Content-Type"); contentType != "application/soap+xml" {
			t.Errorf("expected Content-Type application/soap+xml, got %q", contentType)
		}
		expectedBody := `<soap:Envelope><soap:Body><response>soap success</response></soap:Body></soap:Envelope>`
		actualBody := strings.TrimSpace(w.Body.String())
		if actualBody != expectedBody {
			t.Errorf("expected body %q, got %q", expectedBody, actualBody)
		}
	})

	t.Run("custom soap format error", func(t *testing.T) {
		w := httptest.NewRecorder()
		err := errors.New("soap connection failed")
		ReturnResponse(w, nil, http.StatusInternalServerError, err, nil, "soap")

		if contentType := w.Header().Get("Content-Type"); contentType != "application/soap+xml" {
			t.Errorf("expected Content-Type application/soap+xml, got %q", contentType)
		}
		expectedBody := `<soap:Envelope><soap:Body><soap:Fault><faultstring>soap connection failed</faultstring></soap:Fault></soap:Body></soap:Envelope>`
		actualBody := strings.TrimSpace(w.Body.String())
		if actualBody != expectedBody {
			t.Errorf("expected body %q, got %q", expectedBody, actualBody)
		}
	})
}

// TestHandlerWrapper tests that Handler wrapper converts APIHandler to standard http.HandlerFunc correctly.
func TestHandlerWrapper(t *testing.T) {
	t.Run("wrapper success JSON", func(t *testing.T) {
		h := func(w http.ResponseWriter, r *http.Request) (int, error, interface{}) {
			return http.StatusOK, nil, map[string]string{"result": "wrapped"}
		}
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		Handler(h)(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
		expectedBody := `{"result":"wrapped"}`
		actualBody := strings.TrimSpace(w.Body.String())
		if actualBody != expectedBody {
			t.Errorf("expected body %q, got %q", expectedBody, actualBody)
		}
	})

	t.Run("wrapper success XML format override", func(t *testing.T) {
		h := func(w http.ResponseWriter, r *http.Request) (int, error, interface{}) {
			return http.StatusOK, nil, "wrapped content"
		}
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		HandlerWithFormat(h, "xml")(w, req)

		if contentType := w.Header().Get("Content-Type"); contentType != "application/xml" {
			t.Errorf("expected Content-Type application/xml, got %q", contentType)
		}
		expectedBody := `<response>wrapped content</response>`
		actualBody := strings.TrimSpace(w.Body.String())
		if actualBody != expectedBody {
			t.Errorf("expected body %q, got %q", expectedBody, actualBody)
		}
	})
}
