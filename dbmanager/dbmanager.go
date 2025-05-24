package dbmanager

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"
)

const arbitrageCSV = "arbitrage.csv"

func InsertArbitrage(
	id float64,
	market, symbol, exchange, side string,
	volume float64,
	timestamp time.Time,
	otherExchange, otherSide string,
	buyPrice, sellPrice, profitRatio float64) {

	// TODO. later to DB
	toCSV(
		id,
		market,
		symbol,
		exchange,
		side,
		volume,
		timestamp,
		otherExchange,
		otherSide,
		buyPrice,
		sellPrice,
		profitRatio,
	)
}

func toCSV(
	id float64,
	market, symbol, exchange, side string,
	volume float64,
	timestamp time.Time,
	otherExchange, otherSide string,
	buyPrice, sellPrice, profitRatio float64) {

	fileExists := true
	if _, err := os.Stat(arbitrageCSV); os.IsNotExist(err) {
		fileExists = false
	}

	f, err := os.OpenFile(arbitrageCSV, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Failed to open arbitrage.csv:", err)
		return
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	if !fileExists {
		headers := []string{
			"timestamp", "time", "id", "trade_type", "market", "symbol", "volume",
			"from_exchange", "from_side", "from_price",
			"to_exchange", "to_side", "to_price",
			"profit_ratio",
		}
		writer.Write(headers)
	}

	// 거래 반대 측 정보 추정
	var fromPrice, toPrice float64
	if side == "sell" {
		fromPrice = sellPrice
		toPrice = buyPrice
	} else {
		fromPrice = buyPrice
		toPrice = sellPrice
	}

	// 행 작성
	row := []string{
		fmt.Sprintf("%d", timestamp.UnixNano()),
		timestamp.Format(time.RFC3339Nano),
		strconv.FormatFloat(id, 'f', 0, 64),
		fmt.Sprintf("%s_%s/%s_%s", exchange, side, otherExchange, otherSide),
		market,
		symbol,
		strconv.FormatFloat(volume, 'f', 8, 64),
		exchange,
		side,
		strconv.FormatFloat(fromPrice, 'f', 2, 64),
		otherExchange,
		otherSide,
		strconv.FormatFloat(toPrice, 'f', 2, 64),
		strconv.FormatFloat(profitRatio, 'f', 2, 64),
	}

	if err := writer.Write(row); err != nil {
		fmt.Println("Failed to write row:", err)
	}
}
