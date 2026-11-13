package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func testServer(t *testing.T) http.Handler {
	t.Helper()
	db, err := sql.Open("sqlite", "file:test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	srv := &server{store: store}
	handler, err := srv.routes()
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func TestCreateAndRedirect(t *testing.T) {
	handler := testServer(t)
	request := httptest.NewRequest(http.MethodPost, "/api/urls", strings.NewReader(`{"url":"https://example.com/path"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d", response.Code, http.StatusCreated)
	}
}

func TestRejectsInvalidURL(t *testing.T) {
	handler := testServer(t)
	request := httptest.NewRequest(http.MethodPost, "/api/urls", strings.NewReader(`{"url":"not a URL"}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid URL status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}
