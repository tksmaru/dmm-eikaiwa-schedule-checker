package app

import (
	"fmt"
	"net/http"
	"regexp"
	"time"
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

	// yyyy-mm-dd HH:MM:ss
	re := regexp.MustCompile("[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01]) ([01][0-9]|2[0-3]):[03]0:00")

	doc, _ := goquery.NewDocumentFromResponse(resp)
	// get all schedule
	doc.Find(".oneday").Each(func(_ int, s *goquery.Selection) {

		date := s.Find(".date").Text() // 受講日
		fmt.Fprintln(w, date)

		s.Find(".bt-open").Each(func(_ int, s2 *goquery.Selection) {

			s3, _ := s2.Attr("id") // 受講可能時刻
			fmt.Fprintln(w, s3)
			dateString := re.FindString(s3)
			fmt.Fprintln(w, dateString)

			const form = "2006-01-02 15:04:06 MST"
			day, _ := time.Parse(form, dateString + " JST")
			fmt.Fprintln(w, day)

		})
	})
}
