package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"sync"
)

var (
	errNotStruct = errors.New("config not struct")
	errNoField   = errors.New("field not found")
)

type config struct {
	Name  string
	Tg    tg
	Pairs map[string]interface{}
}

type tg struct {
	Token    string
	Chat_ids []int
}

var (
	configCache config
	cacheOnce   sync.Once
	cacheMutex  sync.RWMutex
)

func loadConfig() {
	path, _ := os.Getwd()
	file, err := os.Open(path + "/config/config.json")
	if err != nil {
		log.Fatalf("Failed to open config file: %v", err)
	}
	defer file.Close()

	var c config
	err = json.NewDecoder(file).Decode(&c)
	if err != nil {
		log.Fatalf("Failed to decode config file: %v", err)
	}

	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	configCache = c
}

func getCachedConfig(key string) reflect.Value {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	s := reflect.ValueOf(&configCache).Elem()
	if s.Kind() != reflect.Struct {
		log.Fatalln(errNotStruct)
	}

	f := s.FieldByName(key)
	if !f.IsValid() {
		log.Fatalln(errNoField)
	}
	return f
}

func getConfig(key string) interface{} {
	cacheOnce.Do(loadConfig) // 최초 1회만 실행
	return getCachedConfig(key).Interface()
}

func GetName() string {
	return getConfig("Name").(string)
}

func GetTg() tg {
	return getConfig("Tg").(tg)
}

func GetPairsData() map[string]interface{} {
	return getConfig("Pairs").(map[string]interface{})
}

func GetExchanges() []string {
	pairsData := GetPairsData()

	var exchanges []string
	for exchange := range pairsData {
		exchanges = append(exchanges, exchange)
	}
	return exchanges
}

func GetPairs(exchange string) []string {
	pairsData := GetPairsData()

	exchangeData, ok := pairsData[exchange].(map[string]interface{})
	if !ok {
		log.Printf("Exchange %s not found in Pairs", exchange)
		return nil
	}

	var pairs []string
	for market, symbols := range exchangeData {
		symbolsList, ok := symbols.([]interface{})
		if !ok {
			log.Printf("Invalid symbols data for market %s in exchange %s", market, exchange)
			continue
		}
		for _, symbol := range symbolsList {
			symbolStr, ok := symbol.(string)
			if !ok {
				log.Printf("Invalid symbol format for %v in market %s", symbol, market)
				continue
			}
			pairs = append(pairs, fmt.Sprintf("%s:%s", market, symbolStr))
		}
	}

	return pairs
}
