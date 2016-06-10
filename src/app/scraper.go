package app

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/context"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
	"io"
	"regexp"
	"time"
)

const (
	maxDays = 2
	form    = "2006-01-02 15:04:05"
)

type Teacher struct {
	Id      string
	Name    string
	PageUrl string
	IconUrl string
}

// DB
type Lessons struct {
	TeacherId string
	List      []time.Time
	Updated   time.Time
}

func (l *Lessons) GetNotifiableLessons(previous []time.Time) []time.Time {
	notifiable := []time.Time{}
	for _, nowTime := range l.List {
		var notify = true
		for _, prevTime := range previous {
			if nowTime.Equal(prevTime) {
				notify = false
				break
			}
		}
		if notify {
			notifiable = append(notifiable, nowTime)
		}
	}
	return notifiable
}

type TeacherInfo struct {
	Teacher
	Lessons
}

type TeacherInfoError struct {
	*TeacherInfo
	err error
}

// Interface
type Fetcher func(ctx context.Context, url string) (io.ReadCloser, error)

type Now func() time.Time

// impl
func get(ctx context.Context, url string) (io.ReadCloser, error) {

	client := urlfetch.Client(ctx)
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("urlfetch failed. url: %s, context: %v", url, err.Error())
	}
	return resp.Body, nil
}

func now() time.Time {
	return time.Now().In(time.FixedZone("Asia/Tokyo", 9*60*60))
}

type Scraper struct {
	context.Context
	get Fetcher
	now Now
}

func NewScraper(ctx context.Context) *Scraper {
	return &Scraper{
		Context: ctx,
		get:     get,
		now:     now,
	}
}

func (sc *Scraper) GetInfo(id string) (*TeacherInfo, error) {

	url := fmt.Sprintf("http://eikaiwa.dmm.com/teacher/index/%s/", id)

	rc, err := sc.get(sc.Context, url)
	if err != nil {
		return nil, fmt.Errorf("[%s] fetch failed. url: %v, context: %v", id, url, err.Error())
	}
	defer rc.Close()

	doc, err := goquery.NewDocumentFromReader(rc)
	if err != nil {
		return nil, fmt.Errorf("[%s] document creation failed. context: %v", id, err)
	}

	name := doc.Find("h1").Last().Text()

	image, _ := doc.Find(".profile-pic").First().Attr("src")

	available := []time.Time{}
	// yyyy-mm-dd HH:MM:ss
	re := regexp.MustCompile("[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01]) ([01][0-9]|2[0-3]):[03]0:00")

	doc.Find(".oneday").EachWithBreak(func(i int, s *goquery.Selection) bool {
		// 直近のmaxDays日分の予約可能情報を対象とする
		if i >= maxDays {
			return false
		}
		log.Debugf(sc.Context, "[%s] i = %v : %v", id, i, s.Find(".date").Text())

		s.Find(".bt-open").Each(func(_ int, s *goquery.Selection) {

			s2, _ := s.Attr("id") // 受講可能時刻
			dateString := re.FindString(s2)

			day, _ := time.ParseInLocation(form, dateString, time.FixedZone("Asia/Tokyo", 9*60*60))
			log.Debugf(sc.Context, "[%s] parsed date: %v", id, day)

			available = append(available, day)
		})
		return true
	})

	t := &TeacherInfo{}
	t.Teacher = Teacher{
		Id:      id,
		Name:    name,
		PageUrl: url,
		IconUrl: image,
	}
	t.Lessons = Lessons{
		TeacherId: id,
		List:      available,
		Updated:   sc.now(),
	}
	log.Debugf(sc.Context, "[%s] scraped data. Teacher: %v, Lessons: %v", id, t.Teacher, t.Lessons)
	return t, nil

}

func (sc *Scraper) GetInfoAsync(c chan TeacherInfoError, id string) {

	t, err := sc.GetInfo(id)
	if err != nil {
		c <- TeacherInfoError{
			err: err,
		}
	} else {
		c <- TeacherInfoError{
			TeacherInfo: t,
		}
	}
}
