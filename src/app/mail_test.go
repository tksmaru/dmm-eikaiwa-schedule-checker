package app

import (
	"google.golang.org/appengine/aetest"
	"google.golang.org/appengine/mail"
	"reflect"
	"testing"
	"time"
)

func TestNewMail_ShouldSucceed_WithoutAnyErrors(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	m := NewMail(ctx)
	if m.Context == nil {
		t.Fatalf("NewMail should contain context but not. actual: %v", m.Context)
	}
}

func TestComposeMail_ShouldSucceed_WithDefaultMailSenderSettings(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	// Set "mail_send_to"
	reset := setTestEnv("mail_send_to", "hoge@example.com")
	defer reset()

	actual, err := ComposeMail(ctx, getSliceOfInformation())
	if err != nil {
		t.Fatalf("ComposeMail should succeed without any errors. actual error: %s", err.Error())
	}

	expected := &mail.Message{
		Sender:  "DMM Eikaiwa schedule checker <anything@testapp.appspotmail.com>",
		To:      []string{"hoge@example.com"},
		Subject: "[DMM Eikaiwa] upcoming schedule",
		Body:    expectedBody,
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("ComposeMail expected %v but %v: %s", expected, actual)
	}
}

func TestComposeMail_ShouldSucceed_WithAnyMailSenderSettings(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	// Set "mail_send_to"
	reset := setTestEnv("mail_send_to", "hoge@example.com")
	defer reset()

	reset2 := setTestEnv("mail_sender", "hogeadmin@example.com")
	defer reset2()

	actual, err := ComposeMail(ctx, getSliceOfInformation())
	if err != nil {
		t.Fatalf("ComposeMail should succeed without any errors. actual error: %s", err.Error())
	}

	expected := &mail.Message{
		Sender:  "DMM Eikaiwa schedule checker <hogeadmin@example.com>",
		To:      []string{"hoge@example.com"},
		Subject: "[DMM Eikaiwa] upcoming schedule",
		Body:    expectedBody,
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("ComposeMail expected %v but %v: %s", expected, actual)
	}
}

func TestComposeMail_ShouldFail_WhenToNotSet(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	m, err := ComposeMail(ctx, getSliceOfInformation())
	if m != nil {
		t.Fatalf("ComposeMail should fail if ENV value 'to' is not set.: %v", m)
	}

	expected := "Invalid ENV value. to: "
	if err != nil && err.Error() != expected {
		t.Fatalf("ComposeMail should fail if ENV value 'to' is not set. actual error: %s", err.Error())
	}

}

func TestComposeMail_ShouldFail_WhenInformationEmpty(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	reset := setTestEnv("mail_send_to", "hoge@example.com")
	defer reset()

	m, err := ComposeMail(ctx, []Information{})
	if m != nil {
		t.Fatalf("ComposeMail should fail if information is empty.: %v", m)
	}

	expected := "contents has no value. contents: []"
	if err != nil && err.Error() != expected {
		t.Fatalf("ComposeMail should fail if information is empty. actual error: %s", err.Error())
	}
}

func TestComposeMail_ShouldFail_WhenInformationNil(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	reset := setTestEnv("mail_send_to", "hoge@example.com")
	defer reset()

	m, err := ComposeMail(ctx, nil)
	if m != nil {
		t.Fatalf("ComposeMail should fail if information is empty.: %v", m)
	}

	expected := "contents has no value. contents: []"
	if err != nil && err.Error() != expected {
		t.Fatalf("ComposeMail should fail if information is empty. actual error: %s", err.Error())
	}
}

// test helper

func getInformation() Information {
	t := Teacher{
		Id:      "11111",
		Name:    "test_teacher",
		PageUrl: "http://example.com/teacher/",
		IconUrl: "http://example.com/teacher/image.png",
	}
	i := Information{
		Teacher:    t,
		NewLessons: []time.Time{time.Date(2014, time.December, 31, 12, 13, 24, 0, time.UTC)},
	}
	return i
}

func getSliceOfInformation() []Information {
	return []Information{getInformation()}
}

const expectedBody = `
Teacher: test_teacher
2014-12-31(Wed) 12:13:24

Access to http://example.com/teacher/
-------------------------
`
