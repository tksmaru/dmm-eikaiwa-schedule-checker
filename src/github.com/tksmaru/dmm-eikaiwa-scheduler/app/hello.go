package app

import (
	//"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"
	"github.com/PuerkitoBio/goquery"

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

type Notification struct {
	Date     time.Time
	New      bool
}

const maxDays = 2

func init() {
	http.HandleFunc("/check", handler)
}

func handler(w http.ResponseWriter, r *http.Request) {

	ctx := appengine.NewContext(r)
	teacher := os.Getenv("teacher")
	if teacher == "" {
		log.Debugf(ctx, "invalid teacher id: %v", teacher)
		return
	}

	client := urlfetch.Client(ctx)
	resp, err := client.Get("http://eikaiwa.dmm.com/teacher/index/" + teacher + "/")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// yyyy-mm-dd HH:MM:ss
	re := regexp.MustCompile("[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01]) ([01][0-9]|2[0-3]):[03]0:00")

	available := []time.Time{}

	doc, _ := goquery.NewDocumentFromResponse(resp)
	// get all schedule

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

			const form = "2006-01-02 15:04:05"
			day, _ := time.ParseInLocation(form, dateString, time.FixedZone("Asia/Tokyo", 9*60*60))
			log.Debugf(ctx, "%v", day)

			available = append(available, day)
		})
		return true
	})

	// キーでデータ作ってデータベースに格納してみる
	// とりあえず、対象教師のIDをstringIDに放り込んでみる
	key := datastore.NewKey(ctx, "Schedule", teacher, 0, nil)

	var old Schedule
	if err := datastore.Get(ctx, key, &old); err != nil {
		// Entityが空の場合は見逃す
		if err.Error() != "datastore: no such entity" {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	new := Schedule {
		teacher,
		available,
		time.Now().In(time.FixedZone("Asia/Tokyo", 9*60*60)),
	}

	if _, err := datastore.Put(ctx, key, &new); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	notifications := []Notification{}
	for _, newVal := range available {
		noti := Notification {
			newVal,
			true,
		}
		// 新規通知かどうか検証
		for _, oldVal := range old.Date {
			if newVal.Equal(oldVal) {
				noti.New = false
				break
			}
		}
		notifications = append(notifications, noti)
	}
	log.Debugf(ctx, "%v", notifications)


	token := os.Getenv("slack_token")
	if token != "" {
		values := url.Values{}
		values.Add("token", token)
		values.Add("channel", "#general")
		values.Add("as_user", "true")
		values.Add("text", "testtest" + teacher)

		res, error := client.PostForm("https://slack.com/api/chat.postMessage", values)
		if error != nil {
			log.Debugf(ctx, "senderror %v", error)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer res.Body.Close()

		b, err := ioutil.ReadAll(res.Body)
		if err == nil {
			log.Debugf(ctx, "response: %v", string(b))
		}
	}
}

