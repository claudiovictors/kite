package kite

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetWithJson(t *testing.T) {
	app := New()
	app.Get("/", func(req Request, res Response) error {
		return res.WithJson(map[string]string{"message": "Hello, World"})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperava 200, veio %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Hello, World") {
		t.Fatalf("body inesperado: %s", rec.Body.String())
	}
}

func TestRouteParam(t *testing.T) {
	app := New()
	app.Get("/users/:id", func(req Request, res Response) error {
		return res.WithJson(map[string]string{"id": req.Param("id")})
	})

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "42") {
		t.Fatalf("esperava param 42 no body, veio: %s", rec.Body.String())
	}
}

func TestGroupWithPrefix(t *testing.T) {
	app := New()
	api := app.Group("/api")
	api.Get("/status", func(req Request, res Response) error {
		return res.WithText("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "ok" {
		t.Fatalf("esperava 'ok', veio: %s", rec.Body.String())
	}
}

func TestMiddlewareChain(t *testing.T) {
	app := New()
	var order []string

	app.Use(func(next HandlerFunc) HandlerFunc {
		return func(req Request, res Response) error {
			order = append(order, "mw1")
			return next(req, res)
		}
	})
	app.Get("/", func(req Request, res Response) error {
		order = append(order, "handler")
		return res.WithText("done")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if strings.Join(order, ",") != "mw1,handler" {
		t.Fatalf("ordem inesperada: %v", order)
	}
}
