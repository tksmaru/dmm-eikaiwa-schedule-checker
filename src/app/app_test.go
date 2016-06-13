package app

import (
	"google.golang.org/appengine/aetest"
	"os"
	"testing"
	"time"
)

func TestInformation_FormattedTime_ShouldSucceed_WithoutAnyErrors(t *testing.T) {

	date := time.Date(2014, time.December, 31, 12, 13, 24, 0, time.UTC)

	inf := Information{
		NewLessons: []time.Time{date},
	}

	actual := inf.FormattedTime("2006-01-02(Mon) 15:04:05")

	if actual[0] != "2014-12-31(Wed) 12:13:24" {
		t.Fatalf("Time should be formatted as '2006-01-02(Mon) 15:04:05'. actual: %v", actual[0])
	}
}

func TestSendMail_ShouldSucceed_WithoutAnyErrors(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	reset := setTestEnv("mail_send_to", "hoge@example.com")
	defer reset()

	t.Log(os.Getenv("mail_send_to"))

	err = sendMail(ctx, getSliceOfInformation())
	if err != nil {
		t.Fatal("sendMail should succeed. actual: %s", err.Error())
	}
}

func TestSendMail_ShouldFail_WhenToNotSet(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	t.Log(os.Getenv("mail_send_to"))

	err = sendMail(ctx, getSliceOfInformation())
	expected := "failed to compose e-mail message. context: Invalid ENV value. to: "
	if err.Error() != expected {
		t.Fatal("expected %s, but %s", expected, err.Error())
	}
}

// test helper

func setTestEnv(key, val string) func() {
	preVal := os.Getenv(key)
	os.Setenv(key, val)
	return func() {
		os.Setenv(key, preVal)
	}
}
