package websocketmanager

import (
	"github/neosouler7/compass/tgmanager"

	"errors"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	wsMap        = make(map[string]*websocket.Conn)
	mu           sync.Mutex
	ErrReadMsg   = errors.New("reading msg on ws")
	SubscribeMsg = "%s websocket subscribed!\n"
	FilteredMsg  = "%s websocket msg filtered - %s\n"
)

const (
	bmb string = "ws-api.bithumb.com"
	kbt string = "ws-api.korbit.co.kr"
	upb string = "api.upbit.com"
)

type hostPath struct {
	host string
	path string
}

func Conn(exchange string) *websocket.Conn {
	mu.Lock()
	defer mu.Unlock()

	if ws, exists := wsMap[exchange]; exists {
		return ws
	}

	h := &hostPath{}
	h.getHostPath(exchange)

	u := url.URL{Scheme: "wss", Host: h.host, Path: h.path}
	ws, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	tgmanager.HandleErr(exchange, err)

	wsMap[exchange] = ws
	return ws
}

func (h *hostPath) getHostPath(exchange string) {
	switch exchange {
	case "bmb":
		h.host = bmb
		h.path = "/websocket/v1"
	case "kbt":
		h.host = kbt
		h.path = "/v2/public"
	case "upb":
		h.host = upb
		h.path = "/websocket/v1"
	}
}

func SendMsg(exchange, msg string) {
	ws := Conn(exchange)
	if ws == nil {
		tgmanager.HandleErr(exchange, errors.New("websocket connection is nil"))
		return
	}

	mu.Lock()
	defer mu.Unlock()
	err := ws.WriteMessage(websocket.TextMessage, []byte(msg))
	tgmanager.HandleErr(exchange, err)
}

func Pong(exchange string) {
	ws := Conn(exchange)
	if ws == nil {
		tgmanager.HandleErr(exchange, errors.New("websocket connection is nil"))
		return
	}

	mu.Lock()
	defer mu.Unlock()
	err := ws.WriteMessage(websocket.PongMessage, []byte{})
	tgmanager.HandleErr(exchange, err)
}
