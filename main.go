package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/robfig/cron"
)

var (
	bot *linebot.Client
	err error

	lineID map[string]string
)

func main() {
	bot, err = linebot.New(
		os.Getenv("CHANNEL_SECRET"),
		os.Getenv("CHANNEL_TOKEN"),
	)
	if err != nil {
		log.Fatal(err)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}

	router := gin.Default()
	cron := cron.New()

	cron.AddFunc("0 17 * * * *", func() {
		w := httptest.NewRecorder()
		mockReq, _ := http.NewRequest("POST", "https://notify-17.herokuapp.com/testpush", nil)
		router.ServeHTTP(w, mockReq)
	})

	router.POST("/callback", callback)
	router.POST("/testpush", pushMsg)

	cron.Start()
	router.Run(":" + port)
}

func pushMsg(c *gin.Context) {
	var messages []linebot.SendingMessage

	hour, minute, _ := time.Now().Clock()
	nowtime := fmt.Sprintf("%02v:%02v", hour, minute)
	messages = append(messages, linebot.NewTextMessage("現在時刻 - "+nowtime))

	for _, id := range lineID {
		_, err := bot.PushMessage(id, messages...).Do()
		if err != nil {
			log.Printf(" [linebot] error: %v\n", err.Error())
		}
	}
}

func callback(c *gin.Context) {
	events, err := bot.ParseRequest(c.Request)
	if err != nil {
		if err == linebot.ErrInvalidSignature {
			c.AbortWithStatus(400)
		} else {
			c.AbortWithStatus(500)
		}
		return
	}
	for _, event := range events {
		groupID := event.Source.GroupID
		if groupID != "" {
			if _, ok := lineID[groupID]; !ok {
				log.Printf(" [linebot] join a new group, ID: %v\n", groupID)
				lineID[groupID] = groupID
			}
		}
	}

}