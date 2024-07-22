package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"text/template"
	"time"
)

type ExchangeRate struct {
	Bid string `json:"bid"`
}

const (
	url      = "http://localhost:8080/cotacao"
	fileName = "cotacao.txt"
)

func main() {
	log.Println("[INFO] Starting client")

	ex, err := getExchangeRate()
	if err != nil {
		panic(err)
	}
	createFile(ex)
}

func getExchangeRate() (ExchangeRate, error) {
	log.Println("[INFO] Getting exchange rate")
	var timeout = 300 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ExchangeRate{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Fatalf("[ERROR] Timeout to get data from server, exceeded %v", timeout)
		}
		return ExchangeRate{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ExchangeRate{}, err
	}

	var exchangeRate ExchangeRate
	if err := json.Unmarshal(body, &exchangeRate); err != nil {
		return ExchangeRate{}, err
	}
	log.Println("[INFO] Response body:", exchangeRate)

	return exchangeRate, nil
}

func createFile(e ExchangeRate) {
	log.Println("[INFO] Creating file")

	f, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	t := template.Must(template.New("cotacao").Parse("Dólar:{{.Bid}}"))
	err = t.Execute(f, e)
	if err != nil {
		panic(err)
	}
}
