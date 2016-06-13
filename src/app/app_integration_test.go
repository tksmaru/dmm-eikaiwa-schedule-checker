// +build !ci

package app

import (
	"google.golang.org/appengine/aetest"
	"sync"
	"testing"
)

func TestPostToSlack_ShouldSucceed_WithoutAnyErrors(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	token, channel := loadSlackSettings()

	reset := setTestEnv("slack_token", token)
	defer reset()

	reset = setTestEnv("slack_channel", channel)
	defer reset()

	var wg sync.WaitGroup
	wg.Add(1)
	postToSlack(ctx, getInformation(), &wg)
	wg.Wait()
}

func TestPostToSlack_ShouldFail_WhenTokenNotSet(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	var wg sync.WaitGroup
	wg.Add(1)
	postToSlack(ctx, getInformation(), &wg)
	wg.Wait()
}
