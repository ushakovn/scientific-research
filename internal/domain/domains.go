package domain

import "time"

type Ticker struct {
  Ticker    string    `json:"ticker"`
  Name      string    `json:"name"`
  Locale    string    `json:"locale"`
  Currency  string    `json:"currency"`
  Active    bool      `json:"active"`
  CreatedAt time.Time `json:"created_at"`
  UpdatedAt time.Time `json:"updated_at"`
}

type Stock struct {
  Ticker    string    `json:"ticker"`
  Open      float64   `json:"open"`
  Close     float64   `json:"close"`
  Highest   float64   `json:"highest"`
  Lowest    float64   `json:"lowest"`
  Timestamp int64     `json:"timestamp"`
  Volume    float64   `json:"volume"`
  CreatedAt time.Time `json:"created_at"`
}
