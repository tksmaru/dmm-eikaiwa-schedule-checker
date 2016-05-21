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
	"google.golang.org/appengine/urlfetch"
)

type Schedule struct {
	Teacher string // 先生のID
	Name    string
	Date    []time.Time // 予約可能日時
	Updated time.Time
}

const (
	maxDays = 2
	form    = "2006-01-02 15:04:05"
)

func init() {
	http.HandleFunc("/check", handler)
}

func handler(w http.ResponseWriter, r *http.Request) {

	ctx := appengine.NewContext(r)
	teachers := os.Getenv("teacher")
	if teachers == "" {
		log.Warningf(ctx, "Invalid ENV settings. teacher: %v", teachers)
		return
	}

	ids := strings.Split(teachers, ",")
	log.Debugf(ctx, "teachers: %v", ids)
	for _, id := range ids {
		err := search(ctx, id)
		if err != nil {
			log.Warningf(ctx, "err: %v", err)
		}
	}
}

type Scraped struct {
	Name     string
	Icon     string
	Page     string
	Lessons  []time.Time
}

func scrape(ctx context.Context, url string) (Scraped, error) {

	var s Scraped

	client := urlfetch.Client(ctx)
	resp, err := client.Get(url)
	if err != nil {
		return s, fmt.Errorf("access error: %s, context: %v", url, err)
	}

	doc, _ := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return s, fmt.Errorf("Document creation error: %s, context: %v", url, err)
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
		log.Debugf(ctx, "i = %v : %v", i, s.Find(".date").Text())


		s.Find(".bt-open").Each(func(_ int, s *goquery.Selection) {

			s2, _ := s.Attr("id") // 受講可能時刻
			dateString := re.FindString(s2)

			day, _ := time.ParseInLocation(form, dateString, time.FixedZone("Asia/Tokyo", 9*60*60))
			log.Debugf(ctx, "%v", day)

			available = append(available, day)
		})
		return true
	})

	s = Scraped{
		Page:     url,
		Name:     name,
		Icon:     image,
		Lessons:  available,
	}
	log.Debugf(ctx, "scraped data : %v", s)

	return s, nil
}

func GetNotifiable(now []time.Time, previous []time.Time) []time.Time {
	notifiable := []time.Time{}
	for _, nowTime := range now {
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

func search(ctx context.Context, id string) error {

	url := fmt.Sprintf("http://eikaiwa.dmm.com/teacher/index/%s/", id)

	scrapedInfo, err := scrape(ctx, url)
	if err != nil {
		return fmt.Errorf("scrape error: %s, context: %v", url, err)
	}

	key := datastore.NewKey(ctx, "Schedule", id, 0, nil)

	var old Schedule
	if err := datastore.Get(ctx, key, &old); err != nil {
		// Entityが空の場合は見逃す
		if err.Error() != "datastore: no such entity" {
			return fmt.Errorf("datastore access error: %s, context: %v", id, err)
		}
	}

	new := Schedule{
		id,
		scrapedInfo.Name,
		scrapedInfo.Lessons,
		time.Now().In(time.FixedZone("Asia/Tokyo", 9*60*60)),
	}

	if _, err := datastore.Put(ctx, key, &new); err != nil {
		return fmt.Errorf("datastore access error: %s, context: %v", new.Teacher, err)
	}

	notifications := GetNotifiable(scrapedInfo.Lessons, old.Date)
	log.Debugf(ctx, "notification data: %v, %v", len(notifications), notifications)

	if len(notifications) == 0 {
		return nil
	}

	noti := Information{
		Name:    scrapedInfo.Name,
		Id:      id,
		Page:    url,
		Icon:    scrapedInfo.Icon,
		Lessons: notifications,
	}
	go notify(ctx, noti)
	return nil
}

type Information struct {
	Name    string
	Id      string
	Page    string
	Icon    string
	Lessons []time.Time
}

func (n *Information) FormattedTime(layout string) []string {
	s := []string{}
	for _, time := range n.Lessons {
		s = append(s, time.Format(layout))
	}
	return s
}

func notify(ctx context.Context, noti Information) {
	notiType := os.Getenv("notification_type")
	switch notiType {
	case "slack":
		toSlack(ctx, noti)
	case "mail":
		toMail(ctx, noti)
	default:
		log.Warningf(ctx, "unknown notification type: %v", notiType)
	}
}

func toSlack(ctx context.Context, noti Information) {

	token := os.Getenv("slack_token")
	if token == "" {
		log.Warningf(ctx, "Invalid ENV value. slack_token: %v", token)
		return
	}

	channel := os.Getenv("channel")
	if channel == "" {
		log.Infof(ctx, "Invalid ENV value. Default value '#general' is set. channel: %v", token)
		channel = "#general"
	}

	values := url.Values{}
	values.Add("token", token)
	values.Add("channel", channel)
	values.Add("as_user", "false")
	values.Add("username", fmt.Sprintf("%s from DMM Eikaiwa", noti.Name))
	values.Add("icon_url", noti.Icon)
	values.Add("text", fmt.Sprintf(messageFormat, strings.Join(noti.FormattedTime(form), "\n"), noti.Page))

	client := urlfetch.Client(ctx)
	res, err := client.PostForm("https://slack.com/api/chat.postMessage", values)
	if err != nil {
		log.Debugf(ctx, "noti send error: %s, context: %v", noti.Id, err)
	}
	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err == nil {
		log.Debugf(ctx, "response: %v", string(b))
	}
}

func toMail(ctx context.Context, noti Information) {
	// TODO write code
}

const messageFormat = `
Hi, you can have a lesson below!
%s

Access to <%s>
`
