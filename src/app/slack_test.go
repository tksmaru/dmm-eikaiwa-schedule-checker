package app

import (
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/appengine/aetest"
	"reflect"
	"testing"
)

func TestNewSlack_ShouldSucceed(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	s := NewSlack(ctx, send)

	if s.Context == nil {
		t.Fatalf("slack should contain context. actual: %v", s)
	}
	if s.post == nil {
		t.Fatalf("slack should contain sender. actual: %v", s)
	}
}

func TestComposeMessage_ShouldFail_WhenSlackTokenNotSet(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	m, err := ComposeMessage(ctx, getInformation())
	if m != nil {
		t.Fatalf("ComposeMessage should fail when ENV value 'slack_token' is not set. actual: %v", m)
	}
	expected := "invalid ENV value. slack_token: "
	if err.Error() != expected {
		t.Fatalf("ComposeMessage should fail when ENV value 'slack_token' is not set. actual: %v", err.Error())
	}
}

func TestComposeMessage_ShouldSucceed_WithDefaultSlackChannel(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	reset := setTestEnv("slack_token", "abcdefg")
	defer reset()

	actual, err := ComposeMessage(ctx, getInformation())
	if err != nil {
		t.Fatalf("ComposeMessage should succeed without any error. actual: %v", err.Error())
	}

	expected := &Message{
		Token:    "abcdefg",
		Channel:  "#general",
		AsUser:   false,
		UserName: "test_teacher from DMM Eikaiwa",
		IconUrl:  "http://example.com/teacher/image.png",
		Text:     expectedText,
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("ComposeMessage expected %v, but %v", expected, actual)
	}
}

func TestComposeMessage_ShouldSucceed_WithAnySlackChannel(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	reset := setTestEnv("slack_token", "abcdefg")
	defer reset()

	reset2 := setTestEnv("slack_channel", "#test")
	defer reset2()

	actual, err := ComposeMessage(ctx, getInformation())
	if err != nil {
		t.Fatalf("ComposeMessage should succeed without any error. actual: %v", err.Error())
	}

	expected := &Message{
		Token:    "abcdefg",
		Channel:  "#test",
		AsUser:   false,
		UserName: "test_teacher from DMM Eikaiwa",
		IconUrl:  "http://example.com/teacher/image.png",
		Text:     expectedText,
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("ComposeMessage expected %v, but %v", expected, actual)
	}
}

const expectedText = `
Hi, you can take a lesson below!
2014-12-31(Wed) 12:13:24

Access to <http://example.com/teacher/>
`

// mock
func mockErrorSend(ctx context.Context, m *Message) ([]byte, error) {
	return nil, fmt.Errorf("something went wrong.")
}

func TestSlack_Send_ShouldFail_WithSendError(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	m := &Message{
		Token:    "abcdefg",
		Channel:  "#test",
		AsUser:   false,
		UserName: "test_teacher from DMM Eikaiwa",
		IconUrl:  "http://example.com/teacher/image.png",
		Text:     expectedText,
	}

	res, err := NewSlack(ctx, mockErrorSend).Send(m)
	if res != nil {
		t.Fatalf("Slack_Send should return nil when send fails. actual: %v", res)
	}

	expected := "notification send failed. context: something went wrong."
	if err.Error() != expected {
		t.Fatalf("Slack_Send expected %v, but %v", expected, err.Error())
	}
}

// mock
func mockSuccessSend(ctx context.Context, m *Message) ([]byte, error) {
	return []byte("send success"), nil
}

func TestSlack_Send_ShouldSucceed_WithoutAnyErrors(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	m := &Message{
		Token:    "abcdefg",
		Channel:  "#test",
		AsUser:   false,
		UserName: "test_teacher from DMM Eikaiwa",
		IconUrl:  "http://example.com/teacher/image.png",
		Text:     expectedText,
	}

	b, err := NewSlack(ctx, mockSuccessSend).Send(m)
	if err != nil {
		t.Fatalf("Slack_Send should succeed without any arrors. actual: %v", err.Error())
	}

	expected := "send success"
	if string(b) != expected {
		t.Fatalf("Slack_Send expected %v, but %v", expected, string(b))
	}
}
