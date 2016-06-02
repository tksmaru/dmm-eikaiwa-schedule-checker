package app

import (
	"google.golang.org/appengine/aetest"
	"os"
	"testing"
	"time"
)

func TestLessons_GetNotifiableLessons_NotifiableLessons(t *testing.T) {

	expected := time.Date(2014, time.December, 31, 12, 13, 24, 0, time.UTC)

	l := Lessons{
		TeacherId: "id",
		List:      []time.Time{expected},
	}

	actual := l.GetNotifiableLessons([]time.Time{})

	if len(actual) != 1 {
		t.Fatalf("Notifiable lessons should have one. actual: %v", len(actual))
	}

	if !actual[0].Equal(expected) {
		t.Fatalf("Notifiable lessons should be equal to '2014-12-31 12:13:24.000 UTC'. actual: %v", actual[0])
	}
}

func TestLessons_GetNotifiableLessons_OneNotifiableLessons(t *testing.T) {

	date := time.Date(2014, time.December, 31, 12, 13, 24, 0, time.UTC)
	expected := time.Date(2014, time.December, 31, 13, 13, 24, 0, time.UTC)

	l := Lessons{
		TeacherId: "id",
		List:      []time.Time{date, expected},
	}

	actual := l.GetNotifiableLessons([]time.Time{date})

	if len(actual) != 1 {
		t.Fatalf("Notifiable lessons should have one. actual: %v", len(actual))
	}

	if !actual[0].Equal(expected) {
		t.Fatalf("Notifiable lessons should be equal to '2014-12-31 13:13:24.000 UTC'. actual: %v", actual[0])
	}
}

func TestLessons_GetNotifiableLessons_NoNotifiableLessons(t *testing.T) {

	date := time.Date(2014, time.December, 31, 12, 13, 24, 0, time.UTC)

	l := Lessons{
		TeacherId: "id",
		List:      []time.Time{date},
	}

	actual := l.GetNotifiableLessons([]time.Time{date})

	if len(actual) != 0 {
		t.Fatalf("Notifiable lessons should have none. actual: %v", len(actual))
	}
}

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

func TestSomething(t *testing.T) {
	reset := setTestEnv("mail_send_to", "hoge@example.com")
	defer reset()

	t.Log(os.Getenv("mail_send_to"))

	ctx, _, _ := aetest.NewContext()

	sendMail(ctx, []Information{})
}

// test helper
func setTestEnv(key, val string) func() {
	preVal := os.Getenv(key)
	os.Setenv(key, val)
	return func() {
		os.Setenv(key, preVal)
	}
}
