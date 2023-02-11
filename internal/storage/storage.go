package storage

import (
  "fmt"
  "scientific-research/internal/domain"
  "scientific-research/internal/storage/db"

  sq "github.com/Masterminds/squirrel"

  log "github.com/sirupsen/logrus"
)

const batchSize = 25

type Storage interface {
  PutTicker(ticker *domain.Ticker) error
  PutStock(stock *domain.Stock) error
  QueryStocks(ticker string) ([]*domain.Stock, error)
  CloseConn() error
}

type storage struct {
  dbClient     db.Client
  tickersCache *Cache[*domain.Ticker]
  stocksCache  *Cache[*domain.Stock]
}

func NewStorage() (Storage, error) {
  client := db.NewSqliteClient()

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

func (s *storage) CloseConn() error {
  return s.dbClient.Close()
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

  columns := []string{
    `ticker_id`,
    `locale`,
    `currency`,
    `active`,
    `updated_at`,
  }
  builder := sq.Insert(``).
    Options(`OR IGNORE`).
    Into(`ticker`).
    Columns(columns...)

  for _, ticker := range tickers {
    values := []any{
      ticker.Ticker,
      ticker.Locale,
      ticker.Currency,
      fmt.Sprintf("%t", ticker.Active),
      ticker.UpdatedAt.Format("2006-01-02"),
    }
    builder = builder.Values(values...)
  }
  return s.execInsertQuery(builder)
}

func (s *storage) PutBatchStocks(stocks []*domain.Stock) error {
  if len(stocks) == 0 {
    return fmt.Errorf("stocks batch is empty")
  }

  columns := []string{
    `stock_id`,
    `ticker_name`,
    `open_price`,
    `close_price`,
    `highest_price`,
    `lowest_price`,
    `timestamp__`,
    `volume`,
  }
  builder := sq.Insert(``).
    Options(`OR IGNORE`).
    Into(`stock`).
    Columns(columns...)

  for _, stock := range stocks {
    stockId := fmt.Sprint(stock.Ticker, "_", stock.Timestamp)

    values := []any{
      stockId,
      stock.Ticker,
      stock.Open,
      stock.Close,
      stock.Highest,
      stock.Lowest,
      stock.Timestamp,
      stock.Volume,
    }
    builder = builder.Values(values...)
  }
  return s.execInsertQuery(builder)
}

func (s *storage) QueryStocks(tickerName string) ([]*domain.Stock, error) {
  columns := []string{
    `ticker_name AS tickerName`,
    `open_price AS open`,
    `close_price AS close`,
    `highest_price AS highest`,
    `lowest_price AS lowest`,
    `timestamp__ AS timestamp`,
    `volume`,
  }
  builder := sq.Select(columns...).From(`stock`)

  if tickerName != "" {
    builder = builder.Where(sq.Eq{
      `ticker_name`: tickerName,
    })
  }

  query, args, err := builder.PlaceholderFormat(sq.Question).ToSql()
  if err != nil {
    return nil, fmt.Errorf("cannot form select query: %v", err)
  }
  res, err := s.dbClient.Query(query, args...)
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
      &stock.Volume,
    ); err != nil {
      return nil, fmt.Errorf("cannot scan row result: %v", err)
    }

    stocks = append(stocks, stock)
  }
  return stocks, nil
}

func (s *storage) execInsertQuery(builder sq.InsertBuilder) error {
  query, args, err := builder.PlaceholderFormat(sq.Question).ToSql()
  if err != nil {
    return fmt.Errorf("cannot form insert query: %v", err)
  }
  err = s.dbClient.Exec(query, args...)
  if err != nil {
    return fmt.Errorf("cannot exec insert query: %v", err)
  }
  return nil
}
