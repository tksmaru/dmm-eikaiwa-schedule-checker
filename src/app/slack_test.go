package app

import (
	"testing"
	"google.golang.org/appengine/aetest"
)

func TestNewSlack(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	s := NewSlack(ctx, send)

	if s.Context == nil {
		t.Fatalf("slack should contain context. actual: %v", s)
	}
	if s.post == nil {
		t.Fatalf("slack should contain sender. actual: %v", s)
	}
}

