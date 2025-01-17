package weaver_test

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bitfield/weaver"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/time/rate"
)

func TestCrawlReturnsExpectedResults(t *testing.T) {
	t.Parallel()
	ts := httptest.NewTLSServer(
		http.FileServer(http.Dir("testdata/crawl")),
	)
	defer ts.Close()
	c := weaver.NewChecker()
	c.HTTPClient = ts.Client()
	c.Output = io.Discard
	c.Check(context.Background(), ts.URL)
	want := []weaver.Result{
		{
			Link:     ts.URL + "/",
			Status:   weaver.StatusOK,
			Message:  "200 OK",
			Referrer: "START",
		},
		{
			Link:     ts.URL + "/go_sucks.html",
			Status:   weaver.StatusOK,
			Message:  "200 OK",
			Referrer: ts.URL + "/",
		},
		{
			Link:     ts.URL + "/bogus",
			Status:   weaver.StatusError,
			Message:  "404 Not Found",
			Referrer: ts.URL + "/go_sucks.html",
		},
		{
			Link:     ts.URL + "/rust_rules.html",
			Status:   weaver.StatusError,
			Message:  "404 Not Found",
			Referrer: ts.URL + "/",
		},
		{
			Link:     ts.URL + "/invalid_links.html",
			Status:   weaver.StatusOK,
			Message:  "200 OK",
			Referrer: ts.URL + "/",
		},
		{
			Link:     "httq://invalid_scheme.html",
			Status:   weaver.StatusError,
			Message:  `Get "httq://invalid_scheme.html": unsupported protocol scheme "httq"`,
			Referrer: ts.URL + "/invalid_links.html",
		},
		{
			Link:     "http:// /",
			Status:   weaver.StatusError,
			Message:  `parse "http:// /": invalid character " " in host name`,
			Referrer: ts.URL + "/invalid_links.html",
		},
	}
	got := c.Results()
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestReduceRateLimit_SetsCorrectLimit(t *testing.T) {
	t.Parallel()
	c := weaver.NewChecker()
	c.SetRateLimit(4)
	c.ReduceRateLimit()
	want := rate.Limit(2)
	got := c.RateLimit()
	if want != got {
		t.Errorf("want %.2f, got %.2f", want, got)
	}
}

func TestFailureToVerifyTLSIsRecordedAsWarning(t *testing.T) {
	t.Parallel()
	ts := httptest.NewTLSServer(nil)
	defer ts.Close()
	ts.Config.ErrorLog = log.New(io.Discard, "", 0)
	c := weaver.NewChecker()
	c.Output = io.Discard
	c.Check(context.Background(), ts.URL)
	got := c.Results()
	if len(got) != 1 {
		t.Fatalf("unexpected result set %v", got)
	}
	res := got[0]
	if res.Link != ts.URL+"/" {
		t.Errorf("want URL %q, got %q", ts.URL+"/", res.Link)
	}
	if res.Status != weaver.StatusWarning {
		t.Errorf("want status %q, got %q", weaver.StatusWarning, res.Status)
	}
}
