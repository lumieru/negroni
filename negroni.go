package negroni

import (
	"log"
	"net/http"
	"os"
	"golang.org/x/net/context"
)

// ContextHandler is a http handler interface with context
type ContextHandler interface {
	ServeHTTPC(ctx context.Context, rw http.ResponseWriter, r *http.Request)
}

// ContextHandlerFunc is a function that implements ContextHandler
type ContextHandlerFunc func(ctx context.Context, rw http.ResponseWriter, r *http.Request)

func (chf ContextHandlerFunc) ServeHTTPC(ctx context.Context, rw http.ResponseWriter, r *http.Request) {
	chf(ctx, rw, r)
}

// Handler handler is an interface that objects can implement to be registered to serve as middleware
// in the Negroni middleware stack.
// ServeHTTP should yield to the next middleware in the chain by invoking the next http.HandlerFunc
// passed in.
//
// If the Handler writes to the ResponseWriter, the next http.HandlerFunc should not be invoked.
type Handler interface {
	ServeHTTP(ctx context.Context, rw http.ResponseWriter, r *http.Request, next ContextHandlerFunc)
}

// HandlerFunc is an adapter to allow the use of ordinary functions as Negroni handlers.
// If f is a function with the appropriate signature, HandlerFunc(f) is a Handler object that calls f.
type HandlerFunc func(ctx context.Context, rw http.ResponseWriter, r *http.Request, next ContextHandlerFunc)

func (h HandlerFunc) ServeHTTP(ctx context.Context, rw http.ResponseWriter, r *http.Request, next ContextHandlerFunc) {
	h(ctx, rw, r, next)
}

type middleware struct {
	handler Handler
	next    *middleware
}

func (m middleware) ServeHTTPC(ctx context.Context, rw http.ResponseWriter, r *http.Request) {
	m.handler.ServeHTTP(ctx, rw, r, m.next.ServeHTTPC)
}

// Wrap converts a http.Handler into a negroni.Handler so it can be used as a Negroni
// middleware. The next http.HandlerFunc is automatically called after the Handler
// is executed.
func Wrap(handler http.Handler) Handler {
	return HandlerFunc(func(ctx context.Context, rw http.ResponseWriter, r *http.Request, next ContextHandlerFunc) {
		handler.ServeHTTP(rw, r)
		next(ctx, rw, r)
	})
}

// WrapCH converts a negroni.ContextHandler into a negroni.Handler so it can be used as a Negroni
// middleware. The next negroni.ContextHandlerFunc is automatically called after the Handler
// is executed.
// IMPORTANT!! handler should read ctx only, and should not write to ctx. Anything write to ctx will
// not pass to the next ContextHandlerFunc
func WrapCH(handler ContextHandler) Handler {
	return HandlerFunc(func(ctx context.Context, rw http.ResponseWriter, r *http.Request, next ContextHandlerFunc) {
		handler.ServeHTTPC(ctx, rw, r)
		next(ctx, rw, r)
	})
}

// Negroni is a stack of Middleware Handlers that can be invoked as an http.Handler.
// Negroni middleware is evaluated in the order that they are added to the stack using
// the Use and UseHandler methods.
type Negroni struct {
	middleware middleware
}

// New returns a new Negroni instance with no middleware preconfigured.
func New(handlers ...Handler) *Negroni {
	return &Negroni{
		middleware: build(handlers),
	}
}

func (n *Negroni) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	n.middleware.ServeHTTPC(context.Background(), NewResponseWriter(rw), r)
}

func (n *Negroni) ServeHTTPC(ctx context.Context, rw http.ResponseWriter, r *http.Request) {
	n.middleware.ServeHTTPC(ctx, NewResponseWriter(rw), r)
}

// Use adds a Handler onto the middleware stack. Handlers are invoked in the order they are added to a Negroni.
func (n *Negroni) Use(handler Handler) {
	appendMiddleware(&(n.middleware), handler)
}

// UseFunc adds a Negroni-style handler function onto the middleware stack.
func (n *Negroni) UseFunc(handlerFunc func(ctx context.Context, rw http.ResponseWriter, r *http.Request, next ContextHandlerFunc)) {
	n.Use(HandlerFunc(handlerFunc))
}

// UseHandler adds a http.Handler onto the middleware stack. Handlers are invoked in the order they are added to a Negroni.
func (n *Negroni) UseHandler(handler http.Handler) {
	n.Use(Wrap(handler))
}

// UseHandler adds a http.HandlerFunc-style handler function onto the middleware stack.
func (n *Negroni) UseHandlerFunc(handlerFunc func(rw http.ResponseWriter, r *http.Request)) {
	n.UseHandler(http.HandlerFunc(handlerFunc))
}

// UseHandler adds a negeroni.ContextHandler onto the middleware stack. Handlers are invoked in the order they are added to a Negroni.
func (n *Negroni) UseContextHandler(handler ContextHandler) {
	n.Use(WrapCH(handler))
}

// UseHandler adds a negeroni.ContextHandlerFunc-style handler function onto the middleware stack.
func (n *Negroni) UseContextHandlerFunc(handlerFunc func(ctx context.Context, rw http.ResponseWriter, r *http.Request)) {
	n.UseContextHandler(ContextHandlerFunc(handlerFunc))
}

// Run is a convenience function that runs the negroni stack as an HTTP
// server. The addr string takes the same format as http.ListenAndServe.
func (n *Negroni) Run(addr string) {
	l := log.New(os.Stdout, "[negroni] ", 0)
	l.Printf("listening on %s", addr)
	l.Fatal(http.ListenAndServe(addr, n))
}

// Returns a list of all the handlers in the current Negroni middleware chain.
func (n *Negroni) Handlers() []Handler {
	var handlers []Handler

	curr := &(n.middleware)
	for !isVoidMiddleware(curr) {
		handlers = append(handlers, curr.handler)
		curr = curr.next
	}

	return handlers
}

func build(handlers []Handler) middleware {
	var next middleware

	if len(handlers) == 0 {
		return voidMiddleware()
	} else if len(handlers) > 1 {
		next = build(handlers[1:])
	} else {
		next = voidMiddleware()
	}

	return middleware{handlers[0], &next}
}

func appendMiddleware(m *middleware, h Handler) {
	var pre *middleware
	curr := m
	for !isVoidMiddleware(curr) {
		pre = curr
		curr = curr.next
	}

	if pre == nil {
		m.handler = h
		next := voidMiddleware()
		m.next = &next
	} else {
		pre.next = &middleware{h, curr}
	}
}

func voidMiddleware() middleware {
	return middleware{
		HandlerFunc(func(ctx context.Context, rw http.ResponseWriter, r *http.Request, next ContextHandlerFunc) {}),
		&middleware{},
	}
}

func isVoidMiddleware(m *middleware) bool {
	if m != nil {
		next := m.next
		if next.handler == nil && next.next == nil {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}
