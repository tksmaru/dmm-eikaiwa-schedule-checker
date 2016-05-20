package app

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
	"github.com/PuerkitoBio/goquery"

	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

type Schedule struct {
	Teacher  string      // 先生のID
	Date     []time.Time // 予約可能日時
	Updated  time.Time
}

const (
	maxDays = 2
	form = "2006-01-02 15:04:05"
)

func init() {
	http.HandleFunc("/check", handler)
}

func handler(w http.ResponseWriter, r *http.Request) {

	ctx := appengine.NewContext(r)
	teach := os.Getenv("teacher")
	if teach == "" {
		log.Debugf(ctx, "invalid teacher id: %v", teach)
		return
	}

	teachers := strings.Split(teach, ",")
	log.Debugf(ctx, "teachers: %v", teachers)
	for _, teacher := range teachers {
		err := search(ctx, teacher)
		if err != nil {
			log.Warningf(ctx, "err: %v", err)
		}
	}
}

func search(ctx context.Context, teacher string) error {

	client := urlfetch.Client(ctx)
	site := fmt.Sprintf("http://eikaiwa.dmm.com/teacher/index/%s/", teacher)
	resp, err := client.Get(site)
	if err != nil {
		return fmt.Errorf("access error: %s, context: %v", site, err)
	}

	doc, _ := goquery.NewDocumentFromResponse(resp)
	// get all schedule

	// teacher's name: Second(last) element of document.getElementsByTagName('h1')
	name := doc.Find("h1").Last().Text()
	log.Debugf(ctx, "name : %v", name)

	// teacher's image: document.getElementsByClassName('profile-pic')
	image, _ := doc.Find(".profile-pic").First().Attr("src")
	log.Debugf(ctx, "image : %v", image)

	available := []time.Time{}
	// yyyy-mm-dd HH:MM:ss
	re := regexp.MustCompile("[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01]) ([01][0-9]|2[0-3]):[03]0:00")

	doc.Find(".oneday").EachWithBreak(func(i int, s *goquery.Selection) bool {
		// 直近のmaxDays日分の予約可能情報を対象とする
		log.Debugf(ctx, "i = %v", i)
		if i >= maxDays {
			return false
		}

		// TODO 受講日情報要らない予感
		date := s.Find(".date").Text() // 受講日
		log.Debugf(ctx, "-----%v-----", date)

		s.Find(".bt-open").Each(func(_ int, s *goquery.Selection) {

			s2, _ := s.Attr("id") // 受講可能時刻
			log.Debugf(ctx, "%v", s2)
			dateString := re.FindString(s2)
			log.Debugf(ctx, "%v", dateString)

			day, _ := time.ParseInLocation(form, dateString, time.FixedZone("Asia/Tokyo", 9*60*60))
			log.Debugf(ctx, "%v", day)

			available = append(available, day)
		})
		return true
	})

	key := datastore.NewKey(ctx, "Schedule", teacher, 0, nil)

	var old Schedule
	if err := datastore.Get(ctx, key, &old); err != nil {
		// Entityが空の場合は見逃す
		if err.Error() != "datastore: no such entity" {
			return fmt.Errorf("datastore access error: %s, context: %v", teacher, err)
		}
	}

	new := Schedule {
		teacher,
		available,
		time.Now().In(time.FixedZone("Asia/Tokyo", 9*60*60)),
	}

	if _, err := datastore.Put(ctx, key, &new); err != nil {
		return fmt.Errorf("datastore access error: %s, context: %v", new.Teacher, err)
	}

	notifications := []string{}
	for _, newVal := range available {
		var notify = true
		for _, oldVal := range old.Date {
			if newVal.Equal(oldVal) {
				notify = false
				break
			}
		}
		if notify {
			notifications = append(notifications, newVal.Format(form))
		}
	}
	log.Debugf(ctx, "notification data: %v, %v", len(notifications), notifications)

	if len(notifications) == 0 {
		return nil
	}

	token := os.Getenv("slack_token")
	if token != "" {

		channel := os.Getenv("channel")
		if channel == "" {
			channel = "#general"
		}

		values := url.Values{}
		values.Add("token", token)
		values.Add("channel", channel)
		values.Add("as_user", "false")
		values.Add("username", fmt.Sprintf("%s from DMM Eikaiwa", name))
		values.Add("icon_url", image)
		values.Add("text", fmt.Sprintf(messageFormat, strings.Join(notifications, "\n"), site))

		res, err := client.PostForm("https://slack.com/api/chat.postMessage", values)
		if err != nil {
			log.Debugf(ctx, "senderror %v", err)
			return fmt.Errorf("noti send error: %s, context: %v", teacher, err)
		}
		defer res.Body.Close()

		b, err := ioutil.ReadAll(res.Body)
		if err == nil {
			log.Debugf(ctx, "response: %v", string(b))
		}
	}
	return nil
}

const messageFormat = `
Hi, you can have a lesson below!
%s

Access to <%s>
`
