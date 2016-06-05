package app

import (
	"fmt"
	"golang.org/x/net/context"
	"io/ioutil"
	"net/url"
	"os"
	"google.golang.org/appengine/urlfetch"
	"strings"
	"google.golang.org/appengine/log"
)

// 送信部分のインタフェース
type Sender func(ctx context.Context, values url.Values) ([]byte, error)

type Slack struct {
	context.Context
	Post    Sender
	Content url.Values
}

func (s *Slack) Notify() ([]byte, error) {
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

func NewSlack(ctx context.Context, sender Sender) *Slack {
	return &Slack{
		Context: ctx,
		Post:    sender,
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

const messageFormat = `
Hi, you can take a lesson below!
%s

Access to <%s>
`