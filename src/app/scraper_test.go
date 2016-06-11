package app

import (
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/appengine/aetest"
	"io"
	"os"
	"reflect"
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

func TestNewScraper_ShouldSucceed(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	s := NewScraper(ctx)

	if s.Context == nil {
		t.Fatalf("Scraper should contain context. actual: %v", s.Context)
	}
	if s.get == nil {
		t.Fatalf("NewScraper should contain implimentation of Fetcher. actual: %v", s.get)
	}
	if s.now == nil {
		t.Fatalf("NewScraper should contain implimentation of Now. actual: %v", s.now)
	}
}

// mock
func mockFairFetch(ctx context.Context, url string) (io.ReadCloser, error) {
	return loadDoc("page.html"), nil
}

func mockNow() time.Time {
	return time.Date(2016, time.June, 10, 12, 00, 00, 0, time.FixedZone("Asia/Tokyo", 9*60*60))
}

func TestScraper_GetInfo_ShouldSucceed_WithoutAnyErrors(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	sc := &Scraper{ctx, mockFairFetch, mockNow}
	actual, err := sc.GetInfo("any")
	if err != nil {
		t.Fatalf("Scraper_GetInfo should succeed. actual: %v", err.Error())
	}

	expected := createTeacherInfo()
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("Scraper_GetInfo expected %v, but actual %v", expected, actual)
	}
}

func mockErrorFetch(ctx context.Context, url string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("fetch error")
}

func TestScraper_GetInfo_ShouldFail_WhenFetchFails(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	sc := &Scraper{ctx, mockErrorFetch, mockNow}
	ti, err := sc.GetInfo("any")
	if ti != nil {
		t.Fatalf("Scraper_GetInfo should return nil when send fails. actual: %v", ti)
	}
	expected := "[any] fetch failed. url: http://eikaiwa.dmm.com/teacher/index/any/, context: fetch error"
	if err.Error() != expected {
		t.Fatalf("Scraper_GetInfo expected %v, but %v", expected, err.Error())
	}
}

//func mockInvalidFetch(ctx context.Context, url string) (io.ReadCloser, error) {
//	return loadDoc("invalid.html"), nil
//}
//
//func TestScraper_GetInfo_ShouldFail_WhenInvalidHtml(t *testing.T) {
//	t.Skip()
//
//	ctx, done, err := aetest.NewContext()
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer done()
//
//	sc := &Scraper{ctx, mockInvalidFetch, mockNow}
//	ti, err := sc.GetInfo("any")
//	if ti != nil {
//		t.Fatalf("Scraper_GetInfo should return nil when parse fails. actual: %v", ti)
//	}
//	expected := "[any] document creation failed. context: %v"
//	if err.Error() != expected {
//		t.Fatalf("Scraper_GetInfo expected %v, but %v", expected, err.Error())
//	}
//}

func TestScraper_GetInfoAsync_ShouldSucceed_WithoutAnyErrors(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	c := make(chan TeacherInfoError)
	sc := &Scraper{ctx, mockFairFetch, mockNow}
	go sc.GetInfoAsync(c, "any")
	te := <-c

	if te.err != nil {
		t.Fatalf("Scraper_GetInfoAsync should succeed. actual: %v", te.err.Error())
	}

	expected := createTeacherInfo()
	if !reflect.DeepEqual(te.TeacherInfo, expected) {
		t.Fatalf("Scraper_GetInfoAsync expected %v, but actual %v", expected, te.TeacherInfo)
	}
}

func TestScraper_GetInfoAsync_ShouldFail_WhenFetchFails(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	c := make(chan TeacherInfoError)
	sc := &Scraper{ctx, mockErrorFetch, mockNow}
	go sc.GetInfoAsync(c, "any")
	te := <-c

	if te.TeacherInfo != nil {
		t.Fatalf("Scraper_GetInfoAsync should return nil when send fails. actual: %v", te.TeacherInfo)
	}
	expected := "[any] fetch failed. url: http://eikaiwa.dmm.com/teacher/index/any/, context: fetch error"
	if te.err.Error() != expected {
		t.Fatalf("Scraper_GetInfo expected %v, but %v", expected, te.err.Error())
	}
}

// test helper

func loadDoc(page string) io.ReadCloser {
	var f *os.File
	var e error

	if f, e = os.Open(fmt.Sprintf("../../testdata/%s", page)); e != nil {
		panic(e.Error())
	}
	return f
}

func createTeacherInfo() *TeacherInfo {

	available := []time.Time{}
	available = append(available, time.Date(2016, time.June, 10, 20, 00, 00, 0, time.FixedZone("Asia/Tokyo", 9*60*60)))
	available = append(available, time.Date(2016, time.June, 10, 20, 30, 00, 0, time.FixedZone("Asia/Tokyo", 9*60*60)))
	available = append(available, time.Date(2016, time.June, 10, 21, 00, 00, 0, time.FixedZone("Asia/Tokyo", 9*60*60)))
	available = append(available, time.Date(2016, time.June, 10, 21, 30, 00, 0, time.FixedZone("Asia/Tokyo", 9*60*60)))
	available = append(available, time.Date(2016, time.June, 10, 22, 00, 00, 0, time.FixedZone("Asia/Tokyo", 9*60*60)))
	available = append(available, time.Date(2016, time.June, 10, 22, 30, 00, 0, time.FixedZone("Asia/Tokyo", 9*60*60)))
	available = append(available, time.Date(2016, time.June, 11, 00, 00, 00, 0, time.FixedZone("Asia/Tokyo", 9*60*60)))
	available = append(available, time.Date(2016, time.June, 11, 00, 30, 00, 0, time.FixedZone("Asia/Tokyo", 9*60*60)))
	available = append(available, time.Date(2016, time.June, 11, 01, 30, 00, 0, time.FixedZone("Asia/Tokyo", 9*60*60)))

	t := &TeacherInfo{}
	t.Teacher = Teacher{
		Id:      "any",
		Name:    "Jelo（ジェロ）",
		PageUrl: "http://eikaiwa.dmm.com/teacher/index/any/",
		IconUrl: "http://image.eikaiwa.dmm.com/teacher/11002/1_201604151625.jpg",
	}
	t.Lessons = Lessons{
		TeacherId: "any",
		List:      available,
		Updated:   time.Date(2016, time.June, 10, 12, 00, 00, 0, time.FixedZone("Asia/Tokyo", 9*60*60)),
	}
	return t
}
