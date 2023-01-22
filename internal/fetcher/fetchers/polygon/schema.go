package polygon

import "time"

type tickerResult struct {
  Ticker         string    `json:"ticker"`
  Name           string    `json:"name"`
  Market         string    `json:"market"`
  Locale         string    `json:"locale"`
  Type           string    `json:"type"`
  Active         bool      `json:"active"`
  CurrencyName   string    `json:"currency_name"`
  LastUpdatedUtc time.Time `json:"last_updated_utc"`
}

type tickersResponse struct {
  Results []*tickerResult `json:"results"`
  Status  string          `json:"status"`
  Count   int             `json:"count"`
  NextUrl string          `json:"next_url"`
}

type pricesResult struct {
  Open      float64 `json:"o"`
  Close     float64 `json:"c"`
  Highest   float64 `json:"h"`
  Lowest    float64 `json:"l"`
  Timestamp int64   `json:"t"`
  Volume    float64 `json:"v"`
}

type pricesResponse struct {
  Adjusted     bool            `json:"adjusted"`
  QueryCount   int             `json:"queryCount"`
  RequestId    string          `json:"request_id"`
  StockResults []*pricesResult `json:"results"`
  ResultsCount int             `json:"resultsCount"`
  Status       string          `json:"status"`
  Ticker       string          `json:"ticker"`
}
