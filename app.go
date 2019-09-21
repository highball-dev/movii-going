package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"google.golang.org/appengine/urlfetch"

	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/line/line-bot-sdk-go/linebot/httphandler"
	"google.golang.org/appengine"
)

type PostParam struct {
	ID   string `json:"ID"`
	Name string `json:"Name"`
	Go   string `json:"Go"`
	Out  string `json:"Out"`
}

func main() {
	botHandler, err := initilizeHTTPHandler()
	if err != nil {
		log.Fatal(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	botHandler.HandleEvents(func(events []*linebot.Event, r *http.Request) {
		c := newContext(r)
		bot, err := botHandler.NewClient()
		if err != nil {
			log.Print(err)
			return
		}

		for _, event := range events {
			if event.Type != linebot.EventTypeMessage {
				return
			}

			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				replyText, mode := buildMessage(message.Text)
				profle, err := bot.GetProfile(event.Source.UserID).WithContext(c).Do()
				if err != nil {
					log.Print(err)
					return
				}
				log.Print(profle.UserID)

				param := PostParam{
					ID:   profle.UserID,
					Name: profle.DisplayName,
				}
				jst := time.FixedZone("Asia/Tokyo", 9*60*60)
				nowJST := time.Now().UTC().In(jst)

				const layout = "2006-01-02 15:04:05"
				// TODO: Enum or 定数管理する
				if mode == "出勤" {
					param.Go = nowJST.Format(layout)
				} else if mode == "退勤" {
					param.Out = nowJST.Format(layout)
				}

				paramBytes, err := json.Marshal(param)
				if err != nil {
					log.Print(err)
					return
				}

				httpClient := urlfetch.Client(c)
				req, err := http.NewRequest("POST", "https://script.google.com/macros/s/AKfycbxJui8mUgrl-pyULT2lu4z2AXV-Qrda2LnrJItFXZw1iuHgKy-a/exec", bytes.NewReader(paramBytes))
				if err != nil {
					log.Print(err)
					return
				}

				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", fmt.Sprintf("token %s", "{your token}"))
				log.Print(req)
				resp, err := httpClient.Do(req)
				if err != nil {
					log.Print(err)
					return
				}
				log.Print(resp)

				if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(replyText)).Do(); err != nil {
					log.Print(err)
				}
			}
		}
	})

	http.Handle("/callback", botHandler)
	appengine.Main()
	log.Printf("Listening on port %s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func newContext(r *http.Request) context.Context {
	return appengine.NewContext(r)
}

func initilizeHTTPHandler() (*httphandler.WebhookHandler, error) {
	return httphandler.New(
		os.Getenv("CHANNEL_SECRET"),
		os.Getenv("CHANNEL_TOKEN"),
	)
}

func buildMessage(text string) (string, string) {
	if isContainsLetsWork(text) {
		return "出勤を確認しました！", "出勤"
	}

	if isContainsLetsSurf(text) {
		return "退勤を確認しました！", "退勤"
	}
	return "確認できませんでした", ""
}

func isContainsLetsWork(targetContent string) bool {
	return strings.Contains(targetContent, "出勤")
}

func isContainsLetsSurf(targetContent string) bool {
	return strings.Contains(targetContent, "退勤")
}
