// main.go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Symbol struct {
	Symbol            string `json:"symbol"`
QuoteAsset            string `json:"quoteAsset"`
    Status            string `json:"status"`
    IsSpotTrading    bool   `json:"isSpotTrading"`
    IsMarginTrading   bool   `json:"isMarginTrading"`
    IsLiquidation     bool   `json:"isLiquidation"`
    Filters           []struct {
        FilterType string `json:"filterType"`
    } `json:"filters"`
}

type ExchangeInfo struct {
    Symbols []Symbol `json:"symbols"`
}


func main() {
    resp, err := http.Get("https://api.binance.com/api/v3/exchangeInfo")
    if err != nil {
        fmt.Println("Error fetching data:", err)
        return
    }
    defer resp.Body.Close()

    var exchangeInfo ExchangeInfo
    if err := json.NewDecoder(resp.Body).Decode(&exchangeInfo); err != nil {
        fmt.Println("Error decoding JSON:", err)
        return
    }

    // fmt.Println("symbol", exchangeInfo.Symbols)
    fmt.Println("high risk pair: ")
    for _, symbol := range exchangeInfo.Symbols {
        if symbol.Status == "BREAK" &&  symbol.QuoteAsset == "USDT" && !symbol.IsSpotTrading {
            fmt.Println(symbol.Symbol)
        }
    }
}