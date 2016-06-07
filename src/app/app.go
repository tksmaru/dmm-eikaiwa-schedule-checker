package app

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	maxDays = 2
	form    = "2006-01-02 15:04:05"
)

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

	c := make(chan TeacherInfoError)
	go NewScraper(ctx, get).getInfoAsync(c, id)
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

func postToSlack(ctx context.Context, inf Information, wg *sync.WaitGroup) {

	defer wg.Done()

	message, err := ComposeMessage(ctx, inf)
	if err != nil {
		log.Errorf(ctx, "[%s] message compose error. context: %v", inf.Id, err.Error())
		return
	}

	b, err := NewSlack(ctx, send).Send(message)
	if err != nil {
		log.Errorf(ctx, "[%s] slack notification error. context: %v", inf.Id, err.Error())
		return
	}
	log.Debugf(ctx, "[%s] slack response: %v", inf.Id, string(b))
}

func sendMail(ctx context.Context, contents []Information) {

	msg, err := ComposeMail(ctx, contents)
	if err != nil {
		log.Errorf(ctx, "failed to compose e-mail message. context: %v", err.Error())
		return
	}

	if err := NewMail(ctx).Send(msg); err != nil {
		log.Errorf(ctx, "Couldn't send email: %v", err)
	}
}
