package main

import (
	"fmt"
	"sync"

	"github/neosouler7/compass/bithumb"
	"github/neosouler7/compass/config"
	"github/neosouler7/compass/korbit"
	"github/neosouler7/compass/upbit"
)

func main() {
	fmt.Println("## START compass")

	var wg sync.WaitGroup

	for _, exchange := range config.GetExchanges() {
		ex := exchange
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("Execute:", ex)

			switch ex {
			case "upb":
				upbit.Run(ex)
			case "kbt":
				korbit.Run(ex)
			case "bmb":
				bithumb.Run(ex)
			default:
				fmt.Println("Unsupported exchange:", ex)
			}
		}()
	}

	wg.Wait()
	fmt.Println("## END compass")
}
