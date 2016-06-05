package app

import (
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/mail"
	"os"
	"strings"
)

type Mail struct {
	context.Context
}

func (m *Mail) Send(msg *mail.Message) error {
	return mail.Send(m.Context, msg)
}

func NewMail(ctx context.Context) *Mail {
	return &Mail{
		Context: ctx,
	}
}

func ComposeMail(ctx context.Context, contents []Information) (*mail.Message, error) {

	sender := os.Getenv("mail_sender")
	if sender == "" {
		sender = fmt.Sprintf("anything@%s.appspotmail.com", appengine.AppID(ctx))
		log.Infof(ctx, "ENV value sender is not set. Default value '%s' is used.", sender)
	}
	to := os.Getenv("mail_send_to")
	if to == "" {
		return nil, fmt.Errorf("Invalid ENV value. to: %v", to)
	}

	body := []string{}
	for _, inf := range contents {
		body = append(body, fmt.Sprintf(mailFormat,
			inf.Name,
			strings.Join(inf.FormattedTime(infForm), "\n"),
			inf.PageUrl))
	}

	msg := &mail.Message{
		Sender:  fmt.Sprintf("DMM Eikaiwa schedule checker <%s>", sender),
		To:      []string{to},
		Subject: "[DMM Eikaiwa] upcoming schedule",
		Body:    fmt.Sprint(strings.Join(body, "\n")),
	}
	log.Debugf(ctx, "mail message: %v", msg)

	return msg, nil
}

const mailFormat = `
Teacher: %s
%s

Access to %s
-------------------------
`
