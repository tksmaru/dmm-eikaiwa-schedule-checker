// +build !ci

package app

import (
	"encoding/json"
	"google.golang.org/appengine/aetest"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"testing"
)

func TestSenderImpl_ShouldSucceed_WithoutAnyErrors(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	token, channel := loadSlackSettings()

	m := &Message{
		Token:    token,
		Channel:  channel,
		AsUser:   false,
		UserName: "test_user",
		IconUrl:  "",
		Text:     "test message text",
	}

	b, err := send(ctx, m)
	if err != nil {
		t.Fatalf("send failed. context: %v", err.Error())
	}

	var actual SlackResponse
	err = json.Unmarshal(b, &actual)
	if err != nil {
		t.Fatalf("json unmarchal failed. context: %v", err.Error())
	}
	//check partial parameters
	if !(actual.Ok &&
		actual.Message.Text == m.Text &&
		actual.Message.Username == m.UserName) {
		t.Fatalf("expected %v, but %v.", m, actual)
	}
}

// test helper

type SlackResponse struct {
	Ok      bool   `json:"ok"`
	Channel string `json:"channel"`
	Ts      string `json:"ts"`
	Message struct {
		Text     string `json:"text"`
		Username string `json:"username"`
		BotId    string `json:"bot_id"`
		Type     string `json:"type"`
		SubType  string `json:"subtype"`
		Ts       string `json:"ts"`
	}
}

func loadSlackSettings() (token string, channel string) {
	b := loadYaml()

	m := make(map[interface{}]interface{})
	e := yaml.Unmarshal(b, &m)
	if e != nil {
		panic(e.Error())
	}
	token, ok := m["env_variables"].(map[interface{}]interface{})["slack_token"].(string)
	if !ok {
		panic("set 'slack_token' ENV_VALUE to app.yaml.")
	}
	channel, ok = m["env_variables"].(map[interface{}]interface{})["slack_channel"].(string)
	if !ok {
		panic("set 'slack_channel' ENV_VALUE to app.yaml.")
	}
	return token, channel
}

func loadYaml() []byte {
	var b []byte
	var e error

	if b, e = ioutil.ReadFile("./app.yaml"); e != nil {
		panic(e.Error())
	}
	return b
}
