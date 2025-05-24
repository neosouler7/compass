package tgmanager

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github/neosouler7/compass/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var (
	StampMicro    = "Jan _2 15:04:05.000000"
	errGetBot     = errors.New("tg init failed")
	errSendMsg    = errors.New("tg sendMsg failed")
	errGetUpdates = errors.New("tg getUpdated failed")

	t           *tgbotapi.BotAPI
	once        sync.Once
	b           bot
	mu          sync.Mutex
	errorFile   = "errors.json"
	minInterval = 3 * time.Second // 최소 메시지 전송 간격
	tgMsgCnt    = 0
	tgMsg       = ""
)

type bot struct {
	token    string
	chat_ids []int
	location *time.Location
}

type ErrorLog struct {
	LatestSend int64    `json:"latest_send"`
	Errors     []string `json:"errors"`
}

func InitBot(t string, c_ids []int, l *time.Location) {
	b = bot{t, c_ids, l}
	// go processErrors()
}

func (b *bot) Bot() *tgbotapi.BotAPI {
	once.Do(func() {
		if t == nil {
			tgPointer, err := tgbotapi.NewBotAPI(b.token)
			if err != nil {
				log.Fatalln(errGetBot)
			}
			t = tgPointer
			t.Debug = true
		}
	})
	return t
}

// when need to get chatId
func GetUpdates() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updateChannel, err := b.Bot().GetUpdatesChan(u)
	if err != nil {
		log.Fatalln(errGetUpdates)
	}

	for update := range updateChannel {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}
		fmt.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
	}
}

func SendMsg(tgMsg string) {
	now := time.Now().In(b.location).Format(StampMicro)
	tgMsg = fmt.Sprintf("%s \n\n%s", tgMsg, now)

	for _, chat_id := range b.chat_ids {
		msg := tgbotapi.NewMessage(int64(chat_id), tgMsg)
		_, err := b.Bot().Send(msg)
		if err != nil {
			log.Fatalln(errSendMsg)
		}
	}
}

func HandleErr(exchange string, err error) {
	if err != nil {
		tgMsgCnt += 1
		tgMsg += fmt.Sprintf("## ERROR %s %s\n%s", config.GetName(), exchange, err.Error())
		if tgMsgCnt == 5 {
			SendMsg(tgMsg)
			tgMsgCnt = 0
			tgMsg = ""
			log.Fatalln(err)
		}
	}
}
