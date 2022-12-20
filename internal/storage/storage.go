package storage

import (
  "scientific-research/internal/storage/db"
  "scientific-research/internal/domain"
  "fmt"
  "regexp"
  "strings"
)

var regSep = regexp.MustCompile(`[\n\r\t]`)

type Storage struct {
  db *db.Client
}

func NewStorage() (*Storage, error) {
  client := &db.Client{}
  err := client.Open()
  if err != nil {
    return nil, err
  }
  return &Storage{
    db: client,
  }, nil
}

func sanitizeQuery(query string) string {
  return regSep.ReplaceAllString(query, "")
}

func (s *Storage) PutBatchTickers(tickers []*domain.Ticker) error {
  query := sanitizeQuery(`INSERT OR IGNORE INTO ticker (
    ticker_id,
    locale,
    currency,
    active,
    updated_at
  ) VALUES %s`)

  valueStr := `(?, ?, ?, ?, ?)`

  var (
    valueStrings []string
    valueArgs    []any
    err          error
  )

  for _, ticker := range tickers {
    valueStrings = append(valueStrings, valueStr)
    currentArgs := []any{
      ticker.Ticker,
      ticker.Locale,
      ticker.Currency,
      fmt.Sprintf("%t", ticker.Active),
      ticker.UpdatedAt.Format("2006-01-02"),
    }
    valueArgs = append(valueArgs, currentArgs...)
  }
  query = fmt.Sprintf(query, strings.Join(valueStrings, ","))

  err = s.db.Exec(query, valueArgs...)
  if err != nil {
    return fmt.Errorf("cannot put stock to storage: %v", err)
  }
  return nil
}

func (s *Storage) PutBatchStocks(stocks []*domain.Stock) error {
  if len(stocks) == 0 {
    return fmt.Errorf("batch is empty")
  }

  query := sanitizeQuery(`INSERT OR IGNORE INTO stock (
      stock_id, 
      ticker_name, 
      open_price,
      close_price,
      highest_price,
      lowest_price,
      timestamp__,
      volume
    ) VALUES %s`)

  valueStr := `(?, ?, ?, ?, ?, ?, ?, ?)`

  var (
    valueStrings []string
    valueArgs    []any
    err          error
  )

  for _, stock := range stocks {
    valueStrings = append(valueStrings, valueStr)
    stockId := fmt.Sprint(stock.Ticker, "_", stock.Timestamp)
    currentArgs := []any{
      stockId,
      stock.Ticker,
      stock.Open,
      stock.Close,
      stock.Highest,
      stock.Lowest,
      stock.Timestamp,
      stock.Volume,
    }
    valueArgs = append(valueArgs, currentArgs...)
  }
  query = fmt.Sprintf(query, strings.Join(valueStrings, ","))

  err = s.db.Exec(query, valueArgs...)
  if err != nil {
    return fmt.Errorf("cannot put stock to storage: %v", err)
  }

  return nil
}

func (s *Storage) QueryStocks(ticker string) ([]*domain.Stock, error) {
  query := sanitizeQuery(`SELECT 
    ticker_name AS ticker,
    open_price AS open,
    close_price AS close,
    highest_price AS highest,
    lowest_price AS lowest,
    timestamp__ AS timestamp,
    volume 
  FROM stock %s`)

  var filter string
  if ticker != "" {
    filter = fmt.Sprintf("WHERE ticker_name = '%s'", ticker)
  }
  query = fmt.Sprintf(query, filter)

  res, err := s.db.Query(query)
  if err != nil {
    return nil, fmt.Errorf("query error: %v", err)
  }
  var stocks []*domain.Stock

  for res.Next() {
    stock := &domain.Stock{}

    if err := res.Scan(
      &stock.Ticker,
      &stock.Open,
      &stock.Close,
      &stock.Highest,
      &stock.Lowest,
      &stock.Timestamp,
      &stock.Volume); err != nil {
      return nil, fmt.Errorf("cannot scan row result: %v", err)
    }

    stocks = append(stocks, stock)
  }
  return stocks, nil
}
