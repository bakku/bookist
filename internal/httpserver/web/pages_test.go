package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ── Index ─────────────────────────────────────────────────────────────────────

func TestIndexRendersMyBooksPage(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	body := resp.Body.String()
	for _, expected := range []string{
		`<title>My Books · Bookist</title>`,
		`<link rel="stylesheet" href="/static/app.css">`,
		`<a href="/" aria-current="page">My Books</a>`,
		`<form class="search" role="search" action="/">`,
		`<h1 class="page-title">My Books</h1>`,
		`All shelves · 14 books`,
		`id="view-list" checked`,
		`<li><a href="#" aria-current="page">All <span class="shelf-count">14</span></a></li>`,
		`<h2 class="shelf-heading">Reading in 2026</h2>`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("expected index response to contain %q", expected)
		}
	}
	if strings.Contains(body, "bulma") {
		t.Error("expected index response not to reference Bulma")
	}
}

func TestIndexRendersSampleBooksInListAndCoverViews(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	body := resp.Body.String()
	if got := strings.Count(body, `<li class="book-row">`); got != 14 {
		t.Errorf("expected 14 list rows, got %d", got)
	}
	if got := strings.Count(body, `<li class="cover-card">`); got != 14 {
		t.Errorf("expected 14 cover cards, got %d", got)
	}
	for _, expected := range []string{
		`<a href="#">Dune</a> <span class="book-series">(Dune #1)</span>`,
		`by <a href="#">David Thomas</a>, <a href="#">Andrew Hunt</a>`,
		`On <a href="#">Favourites</a>, <a href="#">To re-read</a>`,
		`<span class="stars" style="--rating: 4.5" role="img" aria-label="Rated 4.5 out of 5"></span>`,
		`<span class="status status-reading">Reading</span>`,
		`<span class="fact-label">Started</span> <time datetime="2026-09-28">Sep 28, 2026</time>`,
		`<div class="book-added"><time datetime="2024-11-30">Nov 30, 2024</time></div>`,
		`<img class="cover" src="https://covers.openlibrary.org/b/id/14565843-M.jpg" alt="" loading="lazy">`,
		`<span class="cover cover-fallback" data-tone="oxblood" aria-hidden="true" lang="de"><span class="cover-title">Faust. Eine Tragödie</span><span class="cover-author">Johann Wolfgang von Goethe</span></span>`,
		`<p class="cover-card-author">David Thomas, Andrew Hunt</p>`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("expected index response to contain %q", expected)
		}
	}
}
