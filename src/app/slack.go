package app

import (
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"strconv"
)

const (
	infForm = "2006-01-02(Mon) 15:04:05"
)

// 送信部分のインタフェース
type Sender func(ctx context.Context, m *Message) ([]byte, error)

type Slack struct {
	context.Context
	post Sender
}

func (s *Slack) Send(m *Message) ([]byte, error) {
	b, err := s.post(s.Context, m)
	if err != nil {
		err = fmt.Errorf("notification send failed. context: %v", err.Error())
		return nil, err
	}
	return b, nil
}

type Message struct {
	Token    string
	Channel  string
	AsUser   bool
	UserName string
	IconUrl  string
	Text     string
}

//
func Compose(ctx context.Context, inf Information) (*Message, error) {

	token := os.Getenv("slack_token")
	if token == "" {
		return nil, fmt.Errorf("invalid ENV value. slack_token: %v", token)
	}
	channel := os.Getenv("slack_channel")
	if channel == "" {
		log.Infof(ctx, "Invalid ENV value. Default value '#general' is set. channel: %v", channel)
		channel = "#general"
	}

	m := &Message{
		Token:    token,
		Channel:  channel,
		AsUser:   false,
		UserName: fmt.Sprintf("%s from DMM Eikaiwa", inf.Name),
		IconUrl:  inf.IconUrl,
		Text:     fmt.Sprintf(messageFormat, strings.Join(inf.FormattedTime(infForm), "\n"), inf.PageUrl),
	}

	return m, nil
}

func NewSlack(ctx context.Context, sender Sender) *Slack {
	return &Slack{
		Context: ctx,
		post:    sender,
	}
}

// Senderの実装
func send(ctx context.Context, m *Message) ([]byte, error) {

	values := url.Values{}
	values.Add("token", m.Token)
	values.Add("channel", m.Channel)
	values.Add("as_user", strconv.FormatBool(m.AsUser))
	values.Add("username", m.UserName)
	values.Add("icon_url", m.IconUrl)
	values.Add("text", m.Text)

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

const messageFormat = `
Hi, you can take a lesson below!
%s

Access to <%s>
`
