package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/robfig/cron"
)

var (
	bot    *linebot.Client
	err    error
	lineID = make(map[string]int)
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
	router.POST("/testpush", notify)

	cron.Start()
	router.Run(":" + port)
}

// notify send time string when it's xx:17
func notify(c *gin.Context) {
	var messages []linebot.SendingMessage
	taipeiZone, err := time.LoadLocation("Asia/Taipei")
	if err != nil {
		log.Printf(" [linebot] timezone err: %v\n", err.Error())
	}
	hour, minute, _ := time.Now().In(taipeiZone).Clock()
	nowtime := fmt.Sprintf("%02v:%02v", hour, minute)
	messages = append(messages, linebot.NewTextMessage("現在時刻 - "+nowtime))

	for id := range lineID {
		if _, err := bot.PushMessage(id, messages...).Do(); err != nil {
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
				lineID[groupID] = 0
			}
		}

		if event.Type == linebot.EventTypeMessage {
			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				if strings.Contains(message.Text, "17") || strings.Contains(message.Text, "一七") {
					lineID[groupID]++
					var msg *linebot.TextMessage
					if lineID[groupID] == 1 {
						msg = linebot.NewTextMessage("17 出現了第 1 次")
					} else {
						str := strconv.Itoa(lineID[groupID])
						msg = linebot.NewTextMessage("17 又出現了第 " + str + " 次")
					}
					if _, err := bot.PushMessage(groupID, msg).Do(); err != nil {
						log.Printf(" [linebot] error: %v\n", err.Error())
					}
				}
			}
		}
	}
}
