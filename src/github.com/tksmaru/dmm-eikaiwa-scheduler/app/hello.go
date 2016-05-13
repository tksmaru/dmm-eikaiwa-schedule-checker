package app

import (
	"fmt"
	"net/http"
	"regexp"
	"time"
	"github.com/PuerkitoBio/goquery"

	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
	"google.golang.org/appengine/datastore"
)

type Schedule struct {
	Teacher  string      // 先生のID
	Date     []time.Time // 予約可能日時
}

func init() {
	http.HandleFunc("/", handler)
}

func handler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	client := urlfetch.Client(ctx)
	resp, err := client.Get("http://eikaiwa.dmm.com/teacher/index/10439/")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// yyyy-mm-dd HH:MM:ss
	re := regexp.MustCompile("[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01]) ([01][0-9]|2[0-3]):[03]0:00")

	available := []time.Time{}

	doc, _ := goquery.NewDocumentFromResponse(resp)
	// get all schedule
	doc.Find(".oneday").Each(func(_ int, s *goquery.Selection) {
		// TODO 直近の3日分のデータがあれば十分

		// TODO 受講日情報要らない予感
		date := s.Find(".date").Text() // 受講日
		fmt.Fprintln(w, date)

		s.Find(".bt-open").Each(func(_ int, s2 *goquery.Selection) {

			s3, _ := s2.Attr("id") // 受講可能時刻
			//fmt.Fprintln(w, s3)
			dateString := re.FindString(s3)
			//fmt.Fprintln(w, dateString)

			const form = "2006-01-02 15:04:06"
			//day, _ := time.Parse(form, dateString + " JST")
			day, _ := time.ParseInLocation(form, dateString, time.FixedZone("Asia/Tokyo", 9*60*60))
//			day = day.In(time.FixedZone("Asia/Tokyo", 9*60*60))
			fmt.Fprintln(w, day)

			available = append(available, day)
		})
	})

	// キーでデータ作ってデータベースに格納してみる
	// とりあえず、対象教師のIDをstringIDに放り込んでみる
	key := datastore.NewKey(ctx, "Schedule", "10439", 0, nil)

	var old Schedule
	if err := datastore.Get(ctx, key, &old); err != nil {
		// Entityが空の場合は見逃す
		if err.Error() != "datastore: no such entity" {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	fmt.Fprintln(w, old)


	schedule := Schedule {
		"10439",
		available,
	}

	if _, err := datastore.Put(ctx, key, &schedule); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 今のTODOが片付いたら次にやること
	// まず最初にデータを取得して、前回との差分だけを抽出するロジックを作成する

}
