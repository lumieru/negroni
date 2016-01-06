package negroni

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

/* Test Helpers */
func expect(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("Expected %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func refute(t *testing.T, a interface{}, b interface{}) {
	if a == b {
		t.Errorf("Did not expect %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func TestNegroniRun(t *testing.T) {
	// just test that Run doesn't bomb
	go New().Run(":3000")
}

func TestNegroniServeHTTP(t *testing.T) {
	result := ""
	response := httptest.NewRecorder()

	n := New()
	n.Use(HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		result += "foo"
		next(rw, r)
		result += "ban"
	}))
	n.UseFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		result += "bar"
		next(rw, r)
		result += "baz"
	})
	n.UseHandler(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		result += "UseHandler"
	}))
	n.UseHandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		result += "bat"
		rw.WriteHeader(http.StatusBadRequest)
	})

	n.ServeHTTP(response, (*http.Request)(nil))

	expect(t, result, "foobarUseHandlerbatbazban")
	expect(t, response.Code, http.StatusBadRequest)
}

func TestServeHTTPResponseWriter(t *testing.T) {
	response := httptest.NewRecorder()

	n := New()
	n.Use(HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		if prw, ok := rw.(*responseWriter); ok {
			if _, ok := prw.ResponseWriter.(ResponseWriter); ok {
				t.Errorf("%s: prw.ResponseWriter should not be ResponseWriter.", r.URL.String())
			}
		} else {
			t.Errorf("%s: rw should be *responseWriter.", r.URL.String())
		}
	}))

	req, _ := http.NewRequest("GET", "http://http.ResponseWriter", nil)
	n.ServeHTTP(response, req)
	req2, _ := http.NewRequest("GET", "http://negroni.ResponseWriter", nil)
	n.ServeHTTP(NewResponseWriter(response), req2)
}

// Ensures that a Negroni middleware chain 
// can correctly return all of its handlers.
func TestHandlers(t *testing.T) {
	response := httptest.NewRecorder()
	n := New()
	handlers := n.Handlers()
	expect(t, 0, len(handlers))

	n.Use(HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		rw.WriteHeader(http.StatusOK)
	}))

	// Expects the length of handlers to be exactly 1 
	// after adding exactly one handler to the middleware chain
	handlers = n.Handlers()
	expect(t, 1, len(handlers))

	// Ensures that the first handler that is in sequence behaves
	// exactly the same as the one that was registered earlier
	handlers[0].ServeHTTP(response, (*http.Request)(nil), nil)
	expect(t, response.Code, http.StatusOK)
}