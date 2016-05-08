package app

import (
	"fmt"
	"net/http"
	"github.com/PuerkitoBio/goquery"

	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
)

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

	doc, _ := goquery.NewDocumentFromResponse(resp)
	// get all schedule
	doc.Find(".oneday").Each(func(_ int, s *goquery.Selection) {
		date := s.Find(".date").Text() // 受講可能日
		fmt.Fprintln(w, date)
		s.Find(".bt-open").Each(func(_ int, s2 *goquery.Selection) {
			fmt.Fprint(w, s2.Text())
			value, _ := s2.Parent().Attr("class") // 予約可能時間
			fmt.Fprintln(w, value)
		})
	})
}
