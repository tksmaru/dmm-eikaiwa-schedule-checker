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
	"sync"
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
		log.Errorf(ctx, "invalid ENV settings. teachers: %v", teachers)
		return
	}

	notiType := os.Getenv("notification_type")
	if notiType == "" {
		log.Errorf(ctx, "invalid ENV settings. notification_type: %v", notiType)
		return
	}

	ids := strings.Split(teachers, ",")
	log.Debugf(ctx, "teachers: %v", ids)

	ic := make(chan Information, 10)
	for _, id := range ids {
		go search(ic, ctx, id)
	}

	switch notiType {
	case "slack":
		var wg sync.WaitGroup
		for range ids {
			inf := <-ic
			if len(inf.NewLessons) == 0 {
				continue
			}
			wg.Add(1)
			go postToSlack(ctx, inf, &wg)
		}
		wg.Wait()

	case "mail":
		mailContents := []Information{}
		for range ids {
			inf := <-ic
			if len(inf.NewLessons) == 0 {
				continue
			}
			mailContents = append(mailContents, inf)
		}
		if len(mailContents) != 0 {
			sendMail(ctx, mailContents)
		}
	}
}

func search(iChan chan Information, ctx context.Context, id string) {

	inf := Information{}

	c := make(chan TeacherInfo)
	go getInfo(c, ctx, id)
	t := <-c

	if t.err != nil {
		log.Errorf(ctx, "[%s] scrape failed. context: %v", id, t.err)
		iChan <- inf
		return
	}

	key := datastore.NewKey(ctx, "Lessons", id, 0, nil)

	var prev Lessons
	if err := datastore.Get(ctx, key, &prev); err != nil {
		// Entityが空の場合は見逃す
		if err.Error() != "datastore: no such entity" {
			log.Errorf(ctx, "[%s] datastore get operation failed: context: %v", id, err)
			iChan <- inf
			return
		}
	}

	if _, err := datastore.Put(ctx, key, &t.Lessons); err != nil {
		log.Errorf(ctx, "[%s] datastore put operation failed. context: %v", id, err)
		iChan <- inf
		return
	}

	notifiable := t.GetNotifiableLessons(prev.List)
	log.Debugf(ctx, "[%s] notification data: size=%v, %v", id, len(notifiable), notifiable)

	if len(notifiable) == 0 {
		iChan <- inf
		return
	}

	iChan <- Information{
		Teacher:    t.Teacher,
		NewLessons: notifiable,
	}
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

// 送信部分のインタフェース
type Sender func(ctx context.Context, values url.Values) ([]byte, error)

// インタフェース
type Notifier interface {
	Notify() ([]byte, error)
	Compose([]Information) error
}

type Slack struct{
	context.Context
	Post Sender
	Content url.Values
}

func (s *Slack) Notify() ([]byte, error){
	b, err := s.Post(s.Context, s.Content)
	if err != nil {
		err = fmt.Errorf("notification send failed. context: %v", err.Error())
		return nil, err
	}
	return b, nil
}

func (s *Slack) Compose(messages []Information) error {

	token := os.Getenv("slack_token")
	if token == "" {
		return fmt.Errorf("invalid ENV value. slack_token: %v", token)
	}
	channel := os.Getenv("slack_channel")
	if channel == "" {
		log.Infof(s.Context, "Invalid ENV value. Default value '#general' is set. channel: %v", channel)
		channel = "#general"
	}
	if len(messages) == 0 {
		return fmt.Errorf("messages must contain one. len: %v", len(messages))
	}
	inf := messages[0]

	values := url.Values{}
	values.Add("token", token)
	values.Add("channel", channel)
	values.Add("as_user", "false")
	values.Add("username", fmt.Sprintf("%s from DMM Eikaiwa", inf.Name))
	values.Add("icon_url", messages[0].IconUrl)
	values.Add("text", fmt.Sprintf(messageFormat, strings.Join(inf.FormattedTime(infForm), "\n"), inf.PageUrl))
	s.Content = values

	return nil
}

func NewSlack(ctx context.Context, sender Sender) *Slack{
	return &Slack{
		Context: ctx,
		Post: sender,
	}
}

// Senderの実装
func send(ctx context.Context, values url.Values) ([]byte, error) {

	client := urlfetch.Client(ctx)
	res, err := client.PostForm("https://slack.com/api/chat.postMessage", values)
	defer res.Body.Close()
	if err != nil {
		err = fmt.Errorf("notification send failed. context: %v", err.Error())
		return nil, err
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		err = fmt.Errorf("response read failure. context: %v", err.Error())
		return nil, err
	}
	return b, nil
}


func postToSlack(ctx context.Context, inf Information, wg *sync.WaitGroup) {

	defer wg.Done()

	slack := NewSlack(ctx, send)
	err := slack.Compose([]Information{inf})
	if err != nil {
		log.Errorf(ctx, "[%s] message compose error. context: %v", inf.Id, err.Error())
		return
	}
	b, err := slack.Notify()
	if err != nil {
		log.Errorf(ctx, "[%s] slack notification error. context: %v", inf.Id, err.Error())
		return
	}
	log.Debugf(ctx, "[%s] slack response: %v", inf.Id, string(b))
}


func sendMail(ctx context.Context, contents []Information) {

	sender := os.Getenv("mail_sender")
	if sender == "" {
		sender = fmt.Sprintf("anything@%s.appspotmail.com", appengine.AppID(ctx))
		log.Infof(ctx, "ENV value sender is not set. Default value '%s' is used.", sender)
	}
	to := os.Getenv("mail_send_to")
	if to == "" {
		log.Errorf(ctx, "Invalid ENV value. to: %v", to)
		return
	}

	body := []string{}
	for _, inf := range contents {
		body = append(body, fmt.Sprintf(mailFormat,
			inf.Name,
			strings.Join(inf.FormattedTime(infForm), "\n"),
			inf.PageUrl))
	}

	msg := &mail.Message{
		Sender:  fmt.Sprintf("DMM Eikaiwa schedule checker <%s>", sender),
		To:      []string{to},
		Subject: "[DMM Eikaiwa] upcoming schedule",
		Body:    fmt.Sprint(strings.Join(body, "\n")),
	}
	log.Debugf(ctx, "mail message: %v", msg)
	if err := mail.Send(ctx, msg); err != nil {
		log.Errorf(ctx, "Couldn't send email: %v", err)
	}
}

const messageFormat = `
Hi, you can take a lesson below!
%s

Access to <%s>
`

const mailFormat = `
Teacher: %s
%s

Access to %s
-------------------------
`
