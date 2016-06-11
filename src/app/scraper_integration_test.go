// +build !ci

package app

import (
	"github.com/PuerkitoBio/goquery"
	"google.golang.org/appengine/aetest"
	"testing"
)

func TestFetcherImpl_ShouldSucceed_WithoutAnyErrors(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	url := "http://eikaiwa.dmm.com/teacher/index/3990/"
	rc, err := get(ctx, url)
	if err != nil {
		t.Fatalf("unsuspected error occured. detail: %v", err.Error())
	}
	defer rc.Close()

	doc, err := goquery.NewDocumentFromReader(rc)
	if err != nil {
		t.Fatalf("unsuspected error occured. detail: %v", err.Error())
	}

	// Just check for teacher's name from the real content.
	name := doc.Find("h1").Last().Text()
	expected := "Nikola V（ニコラ）"
	if name != expected {
		t.Fatalf("expected %s, but %s", expected, name)
	}
}

func TestFetch_ShouldFail_WhenSyntaxError(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	url := "ftp://www.example.com/" // ftp protocol is not supported
	rc, err := get(ctx, url)
	if rc != nil {
		defer rc.Close()
		t.Fatalf("readcloser should be nil when syntax error occuered. actual: %v", rc)
	}

	expected := "urlfetch failed. url: ftp://www.example.com/, context: Get ftp://www.example.com/: API error 1 (urlfetch: INVALID_URL)"
	if err.Error() != expected {
		t.Fatalf("expected [%s], but [%s]", expected, err.Error())
	}
}

func TestFetch_ShouldFail_WhenPageNotExists(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	// from http://d.hatena.ne.jp/ozuma/20120421/1334976694
	url := "http://ozuma.sakura.ne.jp/httpstatus/404" // this page returns 404
	rc, err := get(ctx, url)
	if rc != nil {
		defer rc.Close()
		t.Fatalf("readcloser should be nil when url not exists. actual: %v", rc)
	}

	expected := "request failed. url: http://ozuma.sakura.ne.jp/httpstatus/404, status code: 404"
	if err.Error() != expected {
		t.Fatalf("expected %s, but %s", expected, err.Error())
	}
}

func TestFetch_ShouldFail_WhenPageNotExistsOnDMM(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	url := "http://eikaiwa.dmm.com/teacher/index/_________/"
	rc, err := get(ctx, url)
	if rc != nil {
		defer rc.Close()
		t.Fatalf("readcloser should be nil when url not exists. actual: %v", rc)
	}

	expected := "request redirected to other page.\nexpected: http://eikaiwa.dmm.com/teacher/index/_________/\nactual: http://eikaiwa.dmm.com/"
	if err.Error() != expected {
		t.Fatalf("expected %s, but %s", expected, err.Error())
	}
}
