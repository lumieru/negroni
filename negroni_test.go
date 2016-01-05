package negroni

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"golang.org/x/net/context"
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

type ContextKey int

const (
	KEY_1	ContextKey = 1
	KEY_2	ContextKey = 2
)

func TestNegroniServeHTTP(t *testing.T) {
	result := ""
	response := httptest.NewRecorder()

	n := New()
	n.Use(HandlerFunc(func(ctx context.Context, rw http.ResponseWriter, r *http.Request, next ContextHandlerFunc) {
		result += "foo"
		ctx = context.WithValue(ctx, KEY_1, "key1")
		next(ctx, rw, r)
		expect(t, ctx.Value(KEY_2), nil)
		result += "ban"
	}))
	n.UseFunc(func(ctx context.Context, rw http.ResponseWriter, r *http.Request, next ContextHandlerFunc) {
		result += "bar"
		result += ctx.Value(KEY_1).(string)
		ctx = context.WithValue(ctx, KEY_2, "key2")
		next(ctx, rw, r)
		result += "baz"
	})
	n.UseHandler(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request){
		result += "-UseHandler+"
	}))
	n.UseHandlerFunc(func(rw http.ResponseWriter, r *http.Request){
		result += "-UseHandlerFunc+"
	})
	n.UseContextHandler(ContextHandlerFunc(func(ctx context.Context, rw http.ResponseWriter, r *http.Request){
		result += "-UseContextHandler+"
		result += ctx.Value(KEY_1).(string) + ctx.Value(KEY_2).(string)
	}))
	n.UseContextHandlerFunc(func(ctx context.Context, rw http.ResponseWriter, r *http.Request){
		result += "-UseContextHandlerFunc+"
		result += ctx.Value(KEY_1).(string) + ctx.Value(KEY_2).(string)
	})
	n.Use(HandlerFunc(func(ctx context.Context, rw http.ResponseWriter, r *http.Request, next ContextHandlerFunc) {
		result += "bat"
		result += ctx.Value(KEY_1).(string) + ctx.Value(KEY_2).(string)
		rw.WriteHeader(http.StatusBadRequest)
	}))

	n.ServeHTTP(response, (*http.Request)(nil))

	expect(t, result, "foobarkey1-UseHandler+-UseHandlerFunc+-UseContextHandler+key1key2-UseContextHandlerFunc+key1key2batkey1key2bazban")
	expect(t, response.Code, http.StatusBadRequest)

	result = "test2"
	response = httptest.NewRecorder()

	n.ServeHTTPC(context.Background(), response, (*http.Request)(nil))
	expect(t, result, "test2foobarkey1-UseHandler+-UseHandlerFunc+-UseContextHandler+key1key2-UseContextHandlerFunc+key1key2batkey1key2bazban")
	expect(t, response.Code, http.StatusBadRequest)
}

// Ensures that a Negroni middleware chain 
// can correctly return all of its handlers.
func TestHandlers(t *testing.T) {
	response := httptest.NewRecorder()
	n := New()
	handlers := n.Handlers()
	expect(t, 0, len(handlers))

	n.Use(HandlerFunc(func(ctx context.Context, rw http.ResponseWriter, r *http.Request, next ContextHandlerFunc) {
		rw.WriteHeader(http.StatusOK)
	}))

	// Expects the length of handlers to be exactly 1 
	// after adding exactly one handler to the middleware chain
	handlers = n.Handlers()
	expect(t, 1, len(handlers))

	// Ensures that the first handler that is in sequence behaves
	// exactly the same as the one that was registered earlier
	handlers[0].ServeHTTP(context.TODO(), response, (*http.Request)(nil), nil)
	expect(t, response.Code, http.StatusOK)
}