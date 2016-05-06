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
	resp, err := client.Get("http://eikaiwa.dmm.com/teacher/index/1250/")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
//	fmt.Fprintf(w, "HTTP GET returned status %v", resp.Status)

	doc, _ := goquery.NewDocumentFromResponse(resp)
	// TODO 詳細化
	doc.Find(".oneday").Each(func(_ int, s *goquery.Selection) {
//		url, _ := s.Text()
		fmt.Fprintln(w, s.Text())
	})

	// こいつはうまく動いたのでコンテンツの取得自体はうまくいってる
	//response, err := ioutil.ReadAll(resp.Body)
	//resp.Body.Close()
	//if err != nil {
	//	http.Error(w, err.Error(), http.StatusInternalServerError)
	//	return
	//}
	//fmt.Fprintf(w, "HTTP GET from API call returned: %s", string(response))

}
