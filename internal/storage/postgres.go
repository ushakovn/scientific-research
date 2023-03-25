package storage

import (
  "context"
  "fmt"
  "scientific-research/internal/domain"
  "scientific-research/internal/storage/postgres"

  sq "github.com/Masterminds/squirrel"
  "github.com/jackc/pgx/v4"
  log "github.com/sirupsen/logrus"
)

const (
  tickerBatchSize = 5  // batch size for ticker and ticker details caches
  stocksBatchSize = 25 // batch size for stocks cache
)

type queryBuilder interface {
  ToSql() (string, []any, error)
}

type Storage interface {
  PutTicker(ticker *domain.Ticker) error
  PutTickerDetails(ticker *domain.TickerDetails) error
  PutStock(stock *domain.Stock) error
  PutFetcherState(state *domain.FetcherState) error
  GetFetcherState() (*domain.FetcherState, bool, error)
}

type storage struct {
  ctx                context.Context
  client             postgres.Client
  tickersCache       *Cache[*domain.Ticker]
  tickerDetailsCache *Cache[*domain.TickerDetails]
  stocksCache        *Cache[*domain.Stock]
}

func NewStorage(ctx context.Context, config *postgres.Config) (Storage, error) {
  client, err := postgres.NewClient(ctx, config)
  if err != nil {
    return nil, fmt.Errorf("cannot create new postgres client: %v", err)
  }
  return &storage{
    ctx:                ctx,
    client:             client,
    tickersCache:       NewCache[*domain.Ticker](),
    tickerDetailsCache: NewCache[*domain.TickerDetails](),
    stocksCache:        NewCache[*domain.Stock](),
  }, nil
}

func (s *storage) PutTicker(ticker *domain.Ticker) error {
  cacheSize := s.tickersCache.Size()

  if cacheSize < tickerBatchSize {
    s.tickersCache.Put(ticker)
    log.Infof("put ticker '%s' for company '%s' in cache. current cache size: %d",
      ticker.TickerId, ticker.CompanyName, cacheSize)
    return nil
  }
  tickers := s.tickersCache.Get()

  if err := s.putBatchTickers(tickers); err != nil {
    return err
  }
  log.Infof("put %d tickers in database. flush current cached",
    tickerBatchSize)

  s.tickersCache.Flush()
  return nil
}

func (s *storage) PutStock(stock *domain.Stock) error {
  cacheSize := s.stocksCache.Size()

  if cacheSize < stocksBatchSize {
    s.stocksCache.Put(stock)
    log.Infof("put stock for ticker '%s' in cache. current cache size: %d",
      stock.TickerId, cacheSize+1)
    return nil
  }
  tickers := s.stocksCache.Get()
  if err := s.putBatchStocks(tickers); err != nil {
    return err
  }
  log.Infof("put %d stocks in database. flush current cached",
    stocksBatchSize)

  s.stocksCache.Flush()
  return nil
}

func (s *storage) PutTickerDetails(tickerDetails *domain.TickerDetails) error {
  cacheSize := s.tickerDetailsCache.Size()

  if cacheSize < tickerBatchSize {
    s.tickerDetailsCache.Put(tickerDetails)
    log.Infof("put ticker details for ticker '%s' in cache. current cache size: %d",
      tickerDetails.TickerId, cacheSize+1)
    return nil
  }
  tickersDetails := s.tickerDetailsCache.Get()
  if err := s.putBatchTickerDetails(tickersDetails); err != nil {
    return err
  }
  log.Infof("put %d tickers details in database. flush current cached",
    tickerBatchSize)

  s.stocksCache.Flush()
  return nil
}

func (s *storage) putBatchTickerDetails(tickersDetails []*domain.TickerDetails) error {
  if len(tickersDetails) == 0 {
    return fmt.Errorf("tickers details batch is empty")
  }
  builder := sq.Insert(`ticker_details`).
    Columns(
      `ticker_id`,
      `company_description`,
      `homepage_url`,
      `phone_number`,
      `total_employees`,
      `company_state`,
      `company_city`,
      `company_address`,
      `company_postal_code`,
    )
  for _, details := range tickersDetails {
    builder = builder.Values(
      details.TickerId,
      details.CompanyDescription,
      details.HomepageUrl,
      details.PhoneNumber,
      details.TotalEmployees,
      details.CompanyState,
      details.CompanyCity,
      details.CompanyAddress,
      details.CompanyPostalCode,
    )
  }
  builder = builder.
    Suffix(`ON CONFLICT (ticker_id) DO NOTHING`).
    PlaceholderFormat(sq.Dollar)

  return s.doPutQuery(builder)
}

func (s *storage) putBatchTickers(tickers []*domain.Ticker) error {
  if len(tickers) == 0 {
    return fmt.Errorf("tickers batch is empty")
  }
  builder := sq.Insert(`ticker`).
    Columns(
      `ticker_id`,
      `company_name`,
      `company_locale`,
      `currency_name`,
      `ticker_cik`,
      `active`,
      `created_at`,
      `external_updated_at`,
    )
  for _, ticker := range tickers {
    builder = builder.Values(
      ticker.TickerId,
      ticker.CompanyName,
      ticker.CompanyLocale,
      ticker.CurrencyName,
      ticker.TickerCik,
      ticker.Active,
      ticker.CreatedAt,
      ticker.ExternalUpdatedAt,
    )
  }
  builder = builder.
    Suffix(`ON CONFLICT (ticker_id) DO NOTHING`).
    PlaceholderFormat(sq.Dollar)

  return s.doPutQuery(builder)
}

func (s *storage) putBatchStocks(stocks []*domain.Stock) error {
  if len(stocks) == 0 {
    return fmt.Errorf("stocks batch is empty")
  }
  builder := sq.Insert(`stock`).
    Columns(
      `stock_id`,
      `ticker_id`,
      `open_price`,
      `close_price`,
      `highest_price`,
      `lowest_price`,
      `trading_volume`,
      `stocked_at`,
      `created_at`,
    )
  for _, stock := range stocks {
    builder = builder.Values(
      stock.StockId,
      stock.TickerId,
      stock.OpenPrice,
      stock.ClosePrice,
      stock.HighestPrice,
      stock.LowestPrice,
      stock.TradingVolume,
      stock.StockedAt,
      stock.CreatedAt,
    )
  }
  builder = builder.
    Suffix(`ON CONFLICT (stock_id) DO NOTHING`).
    PlaceholderFormat(sq.Dollar)

  return s.doPutQuery(builder)
}

func (s *storage) doPutQuery(builder queryBuilder) error {
  query, args := mustBuildQuery(builder)
  if _, err := s.client.Exec(s.ctx, query, args...); err != nil {
    return fmt.Errorf("cannot do exec: %v", err)
  }
  return nil
}

func mustBuildQuery(builder queryBuilder) (string, []any) {
  query, args, err := builder.ToSql()
  if err != nil {
    log.Fatalf("build insert query failed: %v", err)
  }
  return query, args
}

func (s *storage) PutFetcherState(state *domain.FetcherState) error {
  builder := sq.Insert(`fetcher_state`).
    Columns(
      // fetcher_state_id is serial type, autoincrement
      `ticker_req_url`,
      `ticker_details_req_url`,
      `stock_req_url`,
      `created_at`,
      `finished`,
    ).
    Values(
      state.TickerReqUrl,
      state.TickerDetailsReqUrl,
      state.StockReqUrl,
      state.CreatedAt,
      state.Finished,
    ).
    PlaceholderFormat(sq.Dollar)

  if err := s.doPutQuery(builder); err != nil {
    return err
  }
  log.Infof("sucessfully put fetcher state in database")
  return nil
}

func (s *storage) GetFetcherState() (*domain.FetcherState, bool, error) {
  const (
    queryLimit = 5
  )
  builder := sq.Select(
    `state_id`,
    `ticker_req_url`,
    `ticker_details_req_url`,
    `stock_req_url`,
    `created_at`,
    `finished`,
  ).
    From(`fetcher_state`).
    OrderBy(`created_at DESC`).
    Limit(queryLimit).
    PlaceholderFormat(sq.Dollar)

  state := &domain.FetcherState{}
  found, err := s.doGetQuery(builder,
    &state.StateId,
    &state.TickerReqUrl,
    &state.TickerDetailsReqUrl,
    &state.StockReqUrl,
    &state.CreatedAt,
    &state.Finished,
  )
  if err != nil {
    return nil, false, err
  }
  if !found {
    return nil, false, nil
  }
  log.Infof("sucessfully get fetcher state from database")
  return state, true, nil
}

func (s *storage) doGetQuery(builder queryBuilder, fields ...any) (bool, error) {
  query, args := mustBuildQuery(builder)
  rows, err := s.client.Query(s.ctx, query, args...)
  if err != nil {
    return false, fmt.Errorf("cannot do query: %v", err)
  }
  return scanFirstQueriedRow(rows, fields)
}

func scanFirstQueriedRow(rows pgx.Rows, fields []any) (bool, error) {
  var hasScannedRow bool
  if rows.Next() {
    if err := rows.Scan(fields...); err != nil {
      return false, fmt.Errorf("cannot scan queried row: %v", err)
    }
    hasScannedRow = true
  }
  return hasScannedRow, nil
}
