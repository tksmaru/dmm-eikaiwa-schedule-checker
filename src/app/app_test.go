package app

import (
	"google.golang.org/appengine/aetest"
	"os"
	"testing"
	"time"
)

func TestInformation_FormattedTime(t *testing.T) {

	date := time.Date(2014, time.December, 31, 12, 13, 24, 0, time.UTC)

	inf := Information{
		NewLessons: []time.Time{date},
	}

	actual := inf.FormattedTime("2006-01-02(Mon) 15:04:05")

	if actual[0] != "2014-12-31(Wed) 12:13:24" {
		t.Fatalf("Time should be formatted as '2006-01-02(Mon) 15:04:05'. actual: %v", actual[0])
	}
}

func TestSendMail(t *testing.T) {
	reset := setTestEnv("mail_send_to", "hoge@example.com")
	defer reset()

	t.Log(os.Getenv("mail_send_to"))

	ctx, _, _ := aetest.NewContext()

	sendMail(ctx, setInformation())
}

// test helper
func setTestEnv(key, val string) func() {
	preVal := os.Getenv(key)
	os.Setenv(key, val)
	return func() {
		os.Setenv(key, preVal)
	}
}
