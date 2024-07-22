package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type ExchangeRate struct {
	Bid string `json:"bid"`
}

type AwesomeApiResponse struct {
	ApiResponse ExchangeRate `json:"USDBRL"`
}

const (
	apiURL = "https://economia.awesomeapi.com.br/json/last/USD-BRL"
)

func main() {
	log.Println("[INFO] Starting server")

	db, err := prepareDB()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/cotacao", getUSDExchangeRate(db))
	http.ListenAndServe(":8080", mux)
}

func prepareDB() (*sql.DB, error) {
	log.Println("[INFO] Preparing DB")

	var err error
	db, err := sql.Open("sqlite3", "./database.db")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
	  CREATE TABLE IF NOT EXISTS exchange_rate (
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
	  bid TEXT
	  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	  );
	`)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func getUSDExchangeRate(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response, err := getExchangeRateFromApi()
		if err != nil {
			panic(err)
		}

		insertExchangeRate(db, response)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func getExchangeRateFromApi() (ExchangeRate, error) {
	log.Println("[INFO] Getting exchange rate from API")
	var timeout = 200 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return ExchangeRate{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Fatalf("[ERROR] Timeout to get data from api, exceeded %v", timeout)
		}
		return ExchangeRate{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ExchangeRate{}, err
	}

	var a AwesomeApiResponse
	if err := json.Unmarshal(body, &a); err != nil {
		return ExchangeRate{}, err
	}

	log.Println("[INFO] Response body:", a.ApiResponse)

	return a.ApiResponse, nil
}

func insertExchangeRate(db *sql.DB, ex ExchangeRate) {
	log.Println("[INFO] Saving exchange rate")
	var timeout = 10 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	stmt, err := db.PrepareContext(ctx, "INSERT INTO exchange_rate (bid) VALUES (?)")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, ex.Bid)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Fatalf("[ERROR] Timeout to insert data, exceeded %v", timeout)
		}
		panic(err)
	}
}
