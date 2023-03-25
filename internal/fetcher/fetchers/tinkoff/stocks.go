package tinkoff // Package tinkoff: unused

import (
  "context"
  "encoding/json"
  "fmt"
  "log"
  "scientific-research/internal/httpclient"
  "sync/atomic"
  "time"

  "golang.org/x/sync/errgroup"
)

type Fetcher struct {
  ctx        context.Context
  client     *httpclient.Client
  totalCount atomic.Uint32
  state      *state
}

func NewFetcher(ctx context.Context) *Fetcher {
  client := httpclient.NewClient()
  return &Fetcher{
    ctx:        ctx,
    client:     client,
    totalCount: atomic.Uint32{},
    state:      &state{},
  }
}

func createStocksPayload(start, end int) ([]byte, error) {
  payload := &stocksPayload{
    PaginationStart: start,
    PaginationEnd:   end,
    StockCountry:    "All",
    OrderType:       "Asc",
    SortType:        "ByName",
  }
  return json.Marshal(payload)
}

func (f *Fetcher) getPaginatedStocksResponse(start, end int) (*stocksResponse, error) {
  payload, err := createStocksPayload(start, end)
  if err != nil {
    return nil, fmt.Errorf("cannot create payload: %v", err)
  }

  resp, err := f.client.Post(stocksAPI, payload)
  if err != nil {
    return nil, fmt.Errorf("cannot load stocks: %v", err)
  }

  stocksResp := &stocksResponse{}
  if err := json.Unmarshal(resp, stocksResp); err != nil {
    return nil, fmt.Errorf("cannot parse json response: %v", err)
  }
  if stocksResp.Status != respStatusOk {
    return nil, fmt.Errorf("bad response loaded: %s", stocksResp.Status)
  }

  return stocksResp, nil
}

func (f *Fetcher) fillStocks(resp *stocksResponse) ([]*Stock, error) {
  var stocks []*Stock

  for _, respStock := range resp.Payload.Values {
    stockT := respStock.Symbol.Type

    if respStock.Symbol.Type != respStockType {
      return nil, fmt.Errorf("fetched value not a stock: %s", stockT)
    }

    stocks = append(stocks, createStock(respStock))
  }
  return stocks, nil
}

func (f *Fetcher) fetchPaginatedStocks() error {
  paginationStart := 0
  paginationEnd := paginationStart

  stocksResp, err := f.getPaginatedStocksResponse(paginationStart, paginationEnd)
  if err != nil {
    return err
  }
  stocksCount := stocksResp.Payload.Total

  g, _ := errgroup.WithContext(f.ctx)
  g.SetLimit(concurrencyLimit)

  for paginationEnd <= stocksCount {
    paginationStart = paginationEnd
    paginationEnd += paginationInc

    g.Go(func() error {
      stocksResp, err := f.getPaginatedStocksResponse(paginationStart, paginationEnd)
      if err != nil {
        return err
      }
      stocksBatch, err := f.fillStocks(stocksResp)
      if err != nil {
        return fmt.Errorf("fill stocks failed: %v", err)
      }
      for range stocksBatch {
        f.countInc(1)
        log.Printf("total stocks received: %d", f.getTotalCount())
      }
      return nil
    })
  }

  return g.Wait()
}

func createStock(resp *stockResp) *Stock {
  symbol := resp.Symbol

  price := &StockPrice{
    Ticker:   symbol.Ticker,
    Value:    resp.Price.Value,
    Currency: resp.Price.Currency,
    Time:     time.Now(),
  }
  var historical []*StockHistoricalPrice

  for _, respPrice := range resp.HistoricalPrices {
    historical = append(historical, &StockHistoricalPrice{
      Ticker: symbol.Ticker,
      Value:  respPrice.Value,
      Time:   respPrice.Time,
    })
  }

  return &Stock{
    Ticker:             symbol.Ticker,
    Name:               symbol.StockName,
    Country:            symbol.Country,
    Logo:               symbol.Logo,
    Price__:            price,
    HistoricalPrices__: historical,
  }
}
