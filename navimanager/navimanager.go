package navimanager

import (
	"fmt"
	"github/neosouler7/compass/commons"
	"github/neosouler7/compass/dbmanager"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	orderbookHistoryMap   = make(map[string][]Orderbook)
	orderbookHistoryMutex sync.RWMutex

	fees_maker = map[string]float64{
		"upb": 0.0005, // 0.05%
		"kbt": 0.0000, // 0.00%
		"bmb": 0.0004, // 0.04%
	}

	fees_taker = map[string]float64{
		"upb": 0.0005, // 0.05%
		"kbt": 0.0005, // 0.05%
		"bmb": 0.0004, // 0.04%
	}

	tradeTTL     = 1 * time.Second
	orderbookTTL = 3 * time.Second

	BUY, SELL = "buy", "sell"

	recentTradeIds      = make(map[float64]time.Time)
	recentTradeIdsMutex sync.Mutex
)

type Trade struct {
	Id        float64
	Exchange  string
	Market    string
	Symbol    string
	Price     float64
	Volume    float64
	Side      string // buy or sell
	Timestamp time.Time
}

type Order struct {
	Price  float64
	Volume float64
}

type Orderbook struct {
	Asks      []Order
	Bids      []Order
	Timestamp time.Time
}

// 수신된 Raw Orderbook → 구조체로 파싱 & 저장 (5초 유지)
func SetOrderbookInfo(exchange string, rJson map[string]interface{}) {
	var market, symbol string
	var askSlice, bidSlice []Order
	var timestamp time.Time

	switch exchange {
	case "upb":
		var pairInfo = strings.Split(rJson["code"].(string), "-")
		market, symbol = strings.ToLower(pairInfo[0]), strings.ToLower(pairInfo[1])
		timestamp, _ = commons.ConvertToTime(rJson["timestamp"].(float64))

		orderbooks := rJson["orderbook_units"].([]interface{})
		for _, ob := range orderbooks {
			o := ob.(map[string]interface{})
			ask := Order{Price: o["ask_price"].(float64), Volume: o["ask_size"].(float64)}
			bid := Order{Price: o["bid_price"].(float64), Volume: o["bid_size"].(float64)}
			askSlice = append(askSlice, ask)
			bidSlice = append(bidSlice, bid)
		}

	case "kbt":
		market = strings.Split(rJson["symbol"].(string), "_")[1]
		symbol = strings.Split(rJson["symbol"].(string), "_")[0]

		rData := rJson["data"]
		timestamp, _ = commons.ConvertToTime(rData.(map[string]interface{})["timestamp"].(float64))

		var askResponse, bidResponse []interface{}
		askResponse = rData.(map[string]interface{})["asks"].([]interface{})
		bidResponse = rData.(map[string]interface{})["bids"].([]interface{})

		for i := 0; i < commons.Min(len(askResponse), len(bidResponse))-1; i++ {
			askR, bidR := askResponse[i].(map[string]interface{}), bidResponse[i].(map[string]interface{})

			askPrice, _ := strconv.ParseFloat(askR["price"].(string), 64)
			askVolume, _ := strconv.ParseFloat(askR["qty"].(string), 64)
			bidPrice, _ := strconv.ParseFloat(bidR["price"].(string), 64)
			bidVolume, _ := strconv.ParseFloat(bidR["qty"].(string), 64)

			ask := Order{Price: askPrice, Volume: askVolume}
			bid := Order{Price: bidPrice, Volume: bidVolume}
			askSlice = append(askSlice, ask)
			bidSlice = append(bidSlice, bid)
		}

	case "bmb":
		var pairInfo = strings.Split(rJson["code"].(string), "-")
		market, symbol = strings.ToLower(pairInfo[0]), strings.ToLower(pairInfo[1])
		timestamp, _ = commons.ConvertToTime(rJson["timestamp"].(float64))

		orderbooks := rJson["orderbook_units"].([]interface{})
		for _, ob := range orderbooks {
			o := ob.(map[string]interface{})
			ask := Order{Price: o["ask_price"].(float64), Volume: o["ask_size"].(float64)}
			bid := Order{Price: o["bid_price"].(float64), Volume: o["bid_size"].(float64)}
			askSlice = append(askSlice, ask)
			bidSlice = append(bidSlice, bid)
		}
	}

	key := fmt.Sprintf("%s-%s-%s", exchange, market, symbol)
	ob := Orderbook{
		Asks:      askSlice,
		Bids:      bidSlice,
		Timestamp: timestamp,
	}

	orderbookHistoryMutex.Lock()
	history := append(orderbookHistoryMap[key], ob)
	cutoff := time.Now().Add(-orderbookTTL)
	var filtered []Orderbook
	for _, o := range history {
		if o.Timestamp.After(cutoff) {
			filtered = append(filtered, o)
		}
	}
	orderbookHistoryMap[key] = filtered
	orderbookHistoryMutex.Unlock()
}

// 수신된 Raw Trade → 구조체로 파싱 후 onTradeMessageReceived로 전달
func SetTradeInfo(exchange string, rJson map[string]interface{}) {
	var market, symbol, side string
	var price, volume, tradeId float64
	var timestamp time.Time

	switch exchange {
	case "upb":
		var pairInfo = strings.Split(rJson["code"].(string), "-")
		market, symbol = strings.ToLower(pairInfo[0]), strings.ToLower(pairInfo[1])

		timestamp, _ = commons.ConvertToTime(rJson["trade_timestamp"].(float64))

		ab := rJson["ask_bid"].(string)
		if ab == "ASK" {
			side = BUY
		} else {
			side = SELL
		}

		tradeId = rJson["sequential_id"].(float64)
		price = rJson["trade_price"].(float64)
		volume = rJson["trade_volume"].(float64)

	case "kbt":
		market = strings.Split(rJson["symbol"].(string), "_")[1]
		symbol = strings.Split(rJson["symbol"].(string), "_")[0]

		rDataList := rJson["data"].([]interface{})
		rData := rDataList[0].(map[string]interface{})

		timestamp, _ = commons.ConvertToTime(rData["timestamp"].(float64))

		ab := rData["isBuyerTaker"].(bool)
		if ab {
			side = BUY
		} else {
			side = SELL
		}

		tradeId = rData["tradeId"].(float64)
		price, _ = strconv.ParseFloat(rData["price"].(string), 64)
		volume, _ = strconv.ParseFloat(rData["qty"].(string), 64)

	case "bmb":
		var pairInfo = strings.Split(rJson["code"].(string), "-")
		market, symbol = strings.ToLower(pairInfo[0]), strings.ToLower(pairInfo[1])

		timestamp, _ = commons.ConvertToTime(rJson["trade_timestamp"].(float64))

		ab := rJson["ask_bid"].(string)
		if ab == "ASK" {
			side = BUY
		} else {
			side = SELL
		}

		tradeId = rJson["sequential_id"].(float64)
		price = rJson["trade_price"].(float64)
		volume = rJson["trade_volume"].(float64)
	}

	// key := fmt.Sprintf("%s-%s-%s", exchange, market, symbol)
	t := Trade{
		Id:        tradeId,
		Exchange:  exchange,
		Market:    market,
		Symbol:    symbol,
		Price:     price,
		Volume:    volume,
		Side:      side,
		Timestamp: timestamp,
	}

	onTradeMessageReceived(t)
}

func getClosestOrderbook(key string, target time.Time) (Orderbook, bool) {
	orderbookHistoryMutex.RLock()
	defer orderbookHistoryMutex.RUnlock()

	history := orderbookHistoryMap[key]
	if len(history) == 0 {
		return Orderbook{}, false
	}

	var closest Orderbook
	var delta time.Duration
	minDelta := time.Duration(math.MaxInt64)
	for _, ob := range history {
		delta = ob.Timestamp.Sub(target)
		if delta < 0 {
			delta = -delta
		}

		if delta < minDelta {
			minDelta = delta
			closest = ob
		}
	}

	return closest, true
}

// tradeId가 duplicate 되는 부분에 대한 방어 코드
func isDuplicateTradeId(id float64) bool {
	recentTradeIdsMutex.Lock()
	defer recentTradeIdsMutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-5 * time.Second) // 5초 이내만 유지

	for k, v := range recentTradeIds {
		if v.Before(cutoff) {
			delete(recentTradeIds, k)
		}
	}

	if _, exists := recentTradeIds[id]; exists {
		return true
	}
	recentTradeIds[id] = now
	return false
}

func GetObTargetPrice(targetVolume float64, obSlice []Order) float64 {
	currentVolume := 0.0
	for _, ob := range obSlice {
		currentVolume += ob.Volume
		if currentVolume >= targetVolume {
			return ob.Price
		}
	}
	return obSlice[len(obSlice)-1].Price
}

// trade 기준 가장 가까운 orderbook 참조 후 차익거래 판단
func onTradeMessageReceived(t Trade) {
	if time.Since(t.Timestamp) > tradeTTL {
		fmt.Printf("%s:%s:%-4s:%-4s - Trade timestamp over TTL\n", t.Exchange, t.Market, t.Symbol, t.Side)
		return
	}

	if isDuplicateTradeId(t.Id) {
		fmt.Printf("%s:%s:%-4s:%-4s:%f - Trade id duplicated\n", t.Exchange, t.Market, t.Symbol, t.Side, t.Id)
		return
	}

	for otherExchange := range fees_maker {
		if t.Exchange == otherExchange {
			continue
		}
		fmt.Printf("%s:%s:%-4s:%-4s - Check chance for %s\n", t.Exchange, t.Market, t.Symbol, t.Side, otherExchange)

		ob, ok := getClosestOrderbook(fmt.Sprintf("%s-%s-%s", otherExchange, t.Market, t.Symbol), t.Timestamp)
		if !ok || len(ob.Asks) == 0 || len(ob.Bids) == 0 {
			continue
		}

		fromFee := fees_maker[t.Exchange] // trade 내역이 maker
		toFee := fees_taker[otherExchange]

		var sellPrice, buyPrice, otherPrice, profitRatio float64
		var otherSide string

		if t.Side == SELL {
			otherSide = BUY
			// otherPrice = ob.asks[0].price
			otherPrice = GetObTargetPrice(t.Volume, ob.Asks)

			sellPrice = t.Price * (1 - fromFee)
			buyPrice = otherPrice * (1 + toFee)

			// profitRatio = (buyPrice - sellPrice) / sellPrice * 100
		} else if t.Side == BUY {
			otherSide = SELL
			otherPrice = GetObTargetPrice(t.Volume, ob.Bids)

			buyPrice = t.Price * (1 + fromFee)
			sellPrice = otherPrice * (1 - toFee)
		}
		profitRatio = (sellPrice - buyPrice) / buyPrice * 100

		if profitRatio >= 0.05 {
			fmt.Println("")
			fmt.Printf("## HIT %s %s -> %f percent\n", t.Market, t.Symbol, profitRatio)
			if t.Side == SELL {
				fmt.Printf("%s %s with %f (-> %f)\n", t.Side, t.Exchange, t.Price, sellPrice)
				fmt.Printf("%s %s with %f (-> %f)\n", otherSide, otherExchange, otherPrice, buyPrice)
			} else if t.Side == BUY {
				fmt.Printf("%s %s with %f (-> %f)\n", t.Side, t.Exchange, t.Price, buyPrice)
				fmt.Printf("%s %s with %f (-> %f)\n", otherSide, otherExchange, otherPrice, sellPrice)
			}
			fmt.Println("")

			dbmanager.InsertArbitrage(t.Id, t.Market, t.Symbol, t.Exchange, t.Side, t.Volume, t.Timestamp, otherExchange, otherSide, buyPrice, sellPrice, profitRatio)
		}
	}
}
