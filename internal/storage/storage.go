package storage

import (
  "fmt"
  "regexp"
  "scientific-research/internal/domain"
  "scientific-research/internal/storage/db"
  "strings"

  log "github.com/sirupsen/logrus"
)

var regSep = regexp.MustCompile(`[\n\r\t]`)

const batchSize = 25

type Storage interface {
  PutTicker(ticker *domain.Ticker) error
  PutStock(stock *domain.Stock) error
  QueryStocks(tickerName string) ([]*domain.Stock, error)
}

type storage struct {
  dbClient     *db.Client
  tickersCache *Cache[*domain.Ticker]
  stocksCache  *Cache[*domain.Stock]
}

func NewStorage() (Storage, error) {
  client := &db.Client{}

  err := client.Open()
  if err != nil {
    return nil, err
  }

  return &storage{
    dbClient:     client,
    tickersCache: NewCache[*domain.Ticker](),
    stocksCache:  NewCache[*domain.Stock](),
  }, nil
}

func sanitizeQuery(query string) string {
  return regSep.ReplaceAllString(query, "")
}

func (s *storage) PutTicker(ticker *domain.Ticker) error {
  cacheSize := s.tickersCache.GetCount()

  if cacheSize < batchSize {
    s.tickersCache.Put(ticker)

    log.Infof("put ticker %s for company %s in cache. current cache size: %d",
      ticker.Ticker, ticker.Name, cacheSize+1)

    return nil
  }
  tickers := s.tickersCache.Get()

  if err := s.PutBatchTickers(tickers); err != nil {
    return err
  }

  log.Infof("put %d tickers in database. flush current cached tickers",
    batchSize)

  s.tickersCache.Flush()

  return nil
}

func (s *storage) PutStock(stock *domain.Stock) error {
  cacheSize := s.stocksCache.GetCount()

  if cacheSize < batchSize {
    s.stocksCache.Put(stock)

    log.Infof("put stock for ticker %s in cache. current cache size: %d",
      stock.Ticker, cacheSize+1)

    return nil
  }
  tickers := s.stocksCache.Get()

  if err := s.PutBatchStocks(tickers); err != nil {
    return err
  }

  log.Infof("put %d stocks in database. flush current cached stocks",
    batchSize)

  s.stocksCache.Flush()

  return nil
}

func (s *storage) PutBatchTickers(tickers []*domain.Ticker) error {
  if len(tickers) == 0 {
    return fmt.Errorf("tickers batch is empty")
  }

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

  err = s.dbClient.Exec(query, valueArgs...)
  if err != nil {
    return fmt.Errorf("cannot put tickers to storage: %v", err)
  }

  return nil
}

func (s *storage) PutBatchStocks(stocks []*domain.Stock) error {
  if len(stocks) == 0 {
    return fmt.Errorf("stocks batch is empty")
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

  err = s.dbClient.Exec(query, valueArgs...)
  if err != nil {
    return fmt.Errorf("cannot put stocks to storage: %v", err)
  }

  return nil
}

func (s *storage) QueryStocks(tickerName string) ([]*domain.Stock, error) {

  query := sanitizeQuery(`SELECT 
    ticker_name AS tickerName,
    open_price AS open,
    close_price AS close,
    highest_price AS highest,
    lowest_price AS lowest,
    timestamp__ AS timestamp,
    volume 
  FROM stock %s`)

  var filter string
  if tickerName != "" {
    filter = fmt.Sprintf("WHERE ticker_name = '%s'", tickerName)
  }
  query = fmt.Sprintf(query, filter)

  res, err := s.dbClient.Query(query)
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
