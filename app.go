package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/net/context"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/taskqueue"
	"google.golang.org/appengine/urlfetch"

	"github.com/joho/godotenv"
	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/line/line-bot-sdk-go/linebot/httphandler"
)

var botHandler *httphandler.WebhookHandler

func init() {
	err := godotenv.Load("line.env")
	if err != nil {
		panic(err)
	}

	botHandler, err = httphandler.New(
		os.Getenv("LINE_BOT_CHANNEL_SECRET"),
		os.Getenv("LINE_BOT_CHANNEL_TOKEN"),
	)
	botHandler.HandleEvents(handleCallback)

	http.Handle("/callback", botHandler)
	http.HandleFunc("/task", handleTask)
}

// Recived Webhook
func handleCallback(evs []*linebot.Event, r *http.Request) {
	c := newContext(r)
	ts := make([]*taskqueue.Task, len(evs))
	for i, e := range evs {
		j, err := json.Marshal(e)
		if err != nil {
			errorf(c, "json.Marshal: %v", err)
			return
		}
		data := base64.StdEncoding.EncodeToString(j)
		t := taskqueue.NewPOSTTask("/task", url.Values{"data": {data}})
		ts[i] = t
	}
	taskqueue.AddMulti(c, ts, "")
}

// Make Messages
func handleTask(w http.ResponseWriter, r *http.Request) {
	c := newContext(r)
	data := r.FormValue("data")
	if data == "" {
		errorf(c, "No data")
		return
	}

	j, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		errorf(c, "base64 DecodeString: %v", err)
		return
	}

	e := new(linebot.Event)
	err = json.Unmarshal(j, e)
	if err != nil {
		errorf(c, "json.Unmarshal: %v", err)
		return
	}

	bot, err := botHandler.NewClient(linebot.WithHTTPClient(urlfetch.Client(c)))
	if err != nil {
		errorf(c, "newLINEBot: %v", err)
		return
	}

	prof := bot.GetProfile(e.Source.UserID)
	p, err := prof.WithContext(c).Do()
	if err != nil {
		errorf(c, "GetProfile: %v", err)
		return
	}
	bot.PushMessage(e.Source.UserID)

	// for Mayu
	// msg := "おはようございます。" + p.DisplayName + "さん。"
	// switch rand.Intn(4) {
	// case 0:
	// 	msg = "おはようございます。" + p.DisplayName + "さん。"
	// case 1:
	// 	msg = p.StatusMessage + "ってどういう意味なんですか？"
	// case 2:
	// 	msg = "あなたのそばにずっと居ますよ💝"
	// case 3:
	// 	msg = "Pさん…💝" + "\n" + p.PictureURL
	// }

	// for Example
	msg := "おはようございます。" + p.DisplayName + "さん。"

	m := linebot.NewTextMessage(msg)

	if _, err = bot.ReplyMessage(e.ReplyToken, m).WithContext(c).Do(); err != nil {
		errorf(c, "ReplayMessage: %v", err)
		return
	}

	w.WriteHeader(200)
}

func newContext(r *http.Request) context.Context {
	return appengine.NewContext(r)
}

func errorf(c context.Context, format string, args ...interface{}) {
	log.Errorf(c, format, args...)
}
