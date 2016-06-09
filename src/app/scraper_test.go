package app

import (
	"testing"
	"time"
)

func TestLessons_GetNotifiableLessons_ShouldSucceed_WithNotifiableLessons(t *testing.T) {

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

func TestLessons_GetNotifiableLessons_ShouldSucceed_WithOneNotifiableLesson(t *testing.T) {

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

func TestLessons_GetNotifiableLessons_ShouldSucceed_WithoutNotifiableLessons(t *testing.T) {

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
