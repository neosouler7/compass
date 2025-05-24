package commons

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github/neosouler7/compass/config"
	"github/neosouler7/compass/tgmanager"
)

// TODO. btc market에서의 eth를 볼 때... 사토시 단위가 짤린다.
// return 값 먼저 확인하고, 하지만 의심가는 부분은 여기. parsing하면서 유실되는 것으로 추정. 10^-8까지 필요.
// func GetObTargetPrice(volume string, orderbook interface{}) string {
// 	/*
// 		ask's price should go up, and bid should go down

// 		ask = [[p1, v1], [p2, v2], [p3, v3] ...]
// 		bid = [[p3, v3], [p2, v2], [p1, p1] ...]
// 	*/
// 	currentVolume := 0.0
// 	targetVolume, err := strconv.ParseFloat(volume, 64)
// 	tgmanager.HandleErr("GetObTargetPrice1", err)

// 	obSlice := orderbook.([]interface{})
// 	for _, ob := range obSlice {
// 		obInfo := ob.([2]string)
// 		volume, err := strconv.ParseFloat(obInfo[1], 64)
// 		tgmanager.HandleErr("GetObTargetPrice2", err)

// 		currentVolume += volume
// 		if currentVolume >= targetVolume {
// 			return obInfo[0]
// 		}
// 	}
// 	return obSlice[len(obSlice)-1].([2]string)[0]
// }

func GetTargetVolume(exchange, market, symbol string) string {
	pairsData := config.GetPairsData()

	exchangeData, _ := pairsData[exchange].(map[string]interface{})
	marketData, _ := exchangeData[market].([]interface{})

	for _, entry := range marketData {
		entryStr, ok := entry.(string)
		if !ok {
			continue
		}

		parts := strings.Split(entryStr, ":")
		if len(parts) != 2 {
			continue
		}

		entrySymbol := parts[0]
		volumeStr := parts[1]

		if entrySymbol == symbol {
			// volume, err := strconv.ParseFloat(volumeStr, 64)
			// if err != nil {
			// 	log.Printf("Invalid volume format for %s in %s/%s", symbol, exchange, market)
			// 	return 0
			// }
			return volumeStr
		}
	}
	return ""
}

func GetPairMap(exchange string) map[string]interface{} {
	pairs := config.GetPairs(exchange)
	m := make(map[string]interface{}, len(pairs)) // 초기 용량 설정

	for _, pair := range pairs {
		idx := strings.Index(pair, ":")
		if idx < 0 {
			log.Printf("Invalid pair format: %s", pair)
			continue
		}

		market := pair[:idx]
		symbol := pair[idx+1:]
		m[symbol+market] = map[string]string{"market": market, "symbol": symbol}
	}
	return m
}

func FormatTs(ts string) string {
	tsLen := len(ts)

	if tsLen < 13 {
		var sb strings.Builder
		sb.WriteString(ts)
		sb.WriteString(strings.Repeat("0", 13-tsLen))
		return sb.String()
	} else if tsLen == 13 { // if millisecond
		return ts
	} else {
		return ts[:13]
	}
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Bytes2Json(data []byte, i interface{}) {
	r := bytes.NewReader(data)
	err := json.NewDecoder(r).Decode(i)
	tgmanager.HandleErr("Bytes2Json", err)
}

func SetTimeZone(name string) *time.Location {
	tz := os.Getenv("TZ")
	if tz == "" {
		tz = "Asia/Seoul"
		fmt.Printf("%s : DEFAULT %s\n", name, tz)
	} else {
		fmt.Printf("%s : SERVER %s\n", name, tz)
	}
	location, _ := time.LoadLocation(tz)
	return location
}

func ConvertToTime(ts interface{}) (time.Time, error) {
	var timestamp float64

	switch v := ts.(type) {
	case float64:
		timestamp = v
	case int:
		timestamp = float64(v)
	case int64:
		timestamp = float64(v)
	case json.Number:
		num, err := v.Float64()
		if err != nil {
			return time.Time{}, fmt.Errorf("json.Number to float64 failed: %w", err)
		}
		timestamp = num
	case string:
		num, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("string to float64 failed: %w", err)
		}
		timestamp = num
	default:
		return time.Time{}, fmt.Errorf("unsupported type %s", reflect.TypeOf(ts).String())
	}

	// Normalize to 13-digit (millisecond) timestamp
	var ms int64
	switch {
	case timestamp > 1e18: // nanoseconds
		ms = int64(timestamp / 1e6)
	case timestamp > 1e15: // microseconds
		ms = int64(timestamp / 1e3)
	case timestamp > 1e12: // already milliseconds
		ms = int64(timestamp)
	case timestamp > 1e9: // seconds with decimal
		ms = int64(timestamp * 1e3)
	case timestamp > 1e8: // assume seconds
		ms = int64(timestamp * 1e3)
	default:
		return time.Time{}, errors.New("timestamp value too small to be valid")
	}

	// Convert to time.Time from milliseconds
	t := time.Unix(0, ms*int64(time.Millisecond))

	return t, nil
}
