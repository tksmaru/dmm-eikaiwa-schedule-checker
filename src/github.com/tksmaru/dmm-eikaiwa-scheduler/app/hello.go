package app

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/mail"
	"google.golang.org/appengine/urlfetch"
)

const (
	maxDays = 2
	form    = "2006-01-02 15:04:05"
	infForm = "2006-01-02(Mon) 15:04:05"
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

// Noti
type Information struct {
	Teacher
	NewLessons []time.Time
}

func (n *Information) FormattedTime(layout string) []string {
	s := []string{}
	for _, time := range n.NewLessons {
		s = append(s, time.Format(layout))
	}
	return s
}

func init() {
	http.HandleFunc("/check", handler)
}

func handler(w http.ResponseWriter, r *http.Request) {

	ctx := appengine.NewContext(r)
	teachers := os.Getenv("teachers")
	if teachers == "" {
		log.Warningf(ctx, "invalid ENV settings. teacher: %v", teachers)
		return
	}

	ids := strings.Split(teachers, ",")
	log.Debugf(ctx, "teachers: %v", ids)

	e := make(chan error, 10)
	for _, id := range ids {
		go search(e, ctx, id)
	}

	for _, id := range ids {
		err := <-e
		if err != nil {
			log.Errorf(ctx, "[%s] operation failed for %s. err: %v", id, id, err)
		} else {
			log.Infof(ctx, "[%s] err: %v", id, err)
		}
	}
}

func search(e chan error, ctx context.Context, id string) {

	c := make(chan TeacherInfo)
	go getInfo(c, ctx, id)
	t := <-c

	if t.err != nil {
		e <- fmt.Errorf("[%s] scrape failed. context: %v", id, t.err)
		return
	}

	key := datastore.NewKey(ctx, "Lessons", id, 0, nil)

	var prev Lessons
	if err := datastore.Get(ctx, key, &prev); err != nil {
		// Entityが空の場合は見逃す
		if err.Error() != "datastore: no such entity" {
			e <- fmt.Errorf("[%s] datastore get operation failed: context: %v", id, err)
			return
		}
	}

	if _, err := datastore.Put(ctx, key, &t.Lessons); err != nil {
		e <- fmt.Errorf("[%s] datastore put operation failed. context: %v", id, err)
		return
	}

	notifiable := t.GetNotifiableLessons(prev.List)
	log.Debugf(ctx, "[%s] notification data: %v, %v", id, len(notifiable), notifiable)

	if len(notifiable) != 0 {
		inf := Information{
			Teacher:    t.Teacher,
			NewLessons: notifiable,
		}
		done := make(chan bool)
		go func(ctx context.Context, inf Information) {
			notiType := os.Getenv("notification_type")
			switch notiType {
			case "slack":
				toSlack(ctx, inf)
			case "mail":
				toMail(ctx, inf)
			default:
				log.Errorf(ctx, "[%s] unknown notification type. notification_type: %v", id, notiType)
			}
			done <- true
		}(ctx, inf)
		<-done
	}
	e <- nil
}

type TeacherInfo struct {
	Teacher
	Lessons
	err error
}

func getInfo(c chan TeacherInfo, ctx context.Context, id string) {

	var t TeacherInfo

	client := urlfetch.Client(ctx)
	url := fmt.Sprintf("http://eikaiwa.dmm.com/teacher/index/%s/", id)

	resp, err := client.Get(url)
	if err != nil {
		t.err = fmt.Errorf("[%s] urlfetch failed. url: %s, context: %v", id, url, err)
		c <- t
		return
	}

	doc, _ := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		t.err = fmt.Errorf("[%s] document creation failed. url: %s, context: %v", id, url, err)
		c <- t
		return
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
		log.Debugf(ctx, "[%s] i = %v : %v", id, i, s.Find(".date").Text())

		s.Find(".bt-open").Each(func(_ int, s *goquery.Selection) {

			s2, _ := s.Attr("id") // 受講可能時刻
			dateString := re.FindString(s2)

			day, _ := time.ParseInLocation(form, dateString, time.FixedZone("Asia/Tokyo", 9*60*60))
			log.Debugf(ctx, "[%s] parsed date: %v", id, day)

			available = append(available, day)
		})
		return true
	})

	t.Teacher = Teacher{
		Id:      id,
		Name:    name,
		PageUrl: url,
		IconUrl: image,
	}
	t.Lessons = Lessons{
		TeacherId: id,
		List:      available,
		Updated:   time.Now().In(time.FixedZone("Asia/Tokyo", 9*60*60)),
	}
	log.Debugf(ctx, "[%s] scraped data. Teacher: %v, Lessons: %v", id, t.Teacher, t.Lessons)
	c <- t
}

func toSlack(ctx context.Context, inf Information) {

	token := os.Getenv("slack_token")
	if token == "" {
		log.Errorf(ctx, "invalid ENV value. slack_token: %v", token)
		return
	}

	channel := os.Getenv("slack_channel")
	if channel == "" {
		log.Infof(ctx, "Invalid ENV value. Default value '#general' is set. channel: %v", channel)
		channel = "#general"
	}

	values := url.Values{}
	values.Add("token", token)
	values.Add("channel", channel)
	values.Add("as_user", "false")
	values.Add("username", fmt.Sprintf("%s from DMM Eikaiwa", inf.Name))
	values.Add("icon_url", inf.IconUrl)
	values.Add("text", fmt.Sprintf(messageFormat, strings.Join(inf.FormattedTime(infForm), "\n"), inf.PageUrl))

	client := urlfetch.Client(ctx)
	res, err := client.PostForm("https://slack.com/api/chat.postMessage", values)
	if err != nil {
		log.Debugf(ctx, "[%s] notification send failed. context: %v", inf.Id, err)
	}
	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err == nil {
		log.Debugf(ctx, "[%s] slack response: %v", inf.Id, string(b))
	}
}

func toMail(ctx context.Context, inf Information) {

	sender := os.Getenv("mail_sender")
	if sender == "" {
		sender = fmt.Sprintf("anything@%s.appspotmail.com", appengine.AppID(ctx))
		log.Infof(ctx, "[%s] ENV value sender is not set. Default value '%s' is used.", inf.Id, sender)
	}
	to := os.Getenv("mail_send_to")
	if to == "" {
		log.Errorf(ctx, "[%s] Invalid ENV value. to: %v", inf.Id, to)
	}

	msg := &mail.Message{
		Sender:  fmt.Sprintf("%s from DMM Eikaiwa <%s>", inf.Name, sender),
		To:      []string{to},
		Subject: "[DMM Eikaiwa] upcoming schedule",
		Body:    fmt.Sprintf(messageFormat, strings.Join(inf.FormattedTime(infForm), "\n"), inf.PageUrl),
	}
	log.Debugf(ctx, "[%s] mail message: %v", inf.Id, msg)
	if err := mail.Send(ctx, msg); err != nil {
		log.Errorf(ctx, "[%s] Couldn't send email: %v", inf.Id, err)
	}
}

const messageFormat = `
Hi, you can take a lesson below!
%s

Access to <%s>
`
