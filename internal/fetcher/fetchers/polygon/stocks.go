package polygon

import (
  "context"
  "encoding/json"
  "fmt"
  "net/url"
  "os"
  "scientific-research/internal/domain"
  "scientific-research/internal/httpclient"
  "scientific-research/internal/storage"
  "sync"
  "time"

  log "github.com/sirupsen/logrus"
)

const (
  clientLimit = 5

  stocksCacheDesc  = "stocks"
  tickersCacheDesc = "tickers"

  cursorKey     = "cursor"
  respStatusOK  = "OK"
  polygonAPIKey = "POLYGON"

  basePrefixAPI = "https://api.polygon.io"
  tickersAPI    = "/v3/reference/tickers"
  pricesAPI     = "/v2/aggs/ticker/%s/range/%d/%s/%s/%s"

  hoursPerYear = 8760
  hoursPerDay  = 24
)

type Fetcher struct {
  ctx          context.Context
  client       *httpclient.Client
  stocksCache  *storage.CacheStorage
  tickersCache *storage.CacheStorage
  token        string
  state        *state
  once         *sync.Once
  ticker       *domain.Ticker
}

func NewFetcher(ctx context.Context) (*Fetcher, error) {
  token, err := getAPIKey()
  if err != nil {
    return nil, err
  }
  client := httpclient.NewClient(ctx, httpclient.WithLimiter(clientLimit))

  fetcherStorage, err := storage.NewStorage()
  if err != nil {
    return nil, err
  }
  stocksCache, err := storage.NewCache(fetcherStorage, stocksCacheDesc)
  if err != nil {
    return nil, err
  }
  tickersCache, err := storage.NewCache(fetcherStorage, tickersCacheDesc)
  if err != nil {
    return nil, err
  }

  return &Fetcher{
    ctx:          ctx,
    client:       client,
    stocksCache:  stocksCache,
    tickersCache: tickersCache,
    token:        token,
    state:        &state{},
    once:         &sync.Once{},
  }, nil
}

func getAPIKey() (string, error) {
  token := os.Getenv(polygonAPIKey)
  if token == "" {
    return "", fmt.Errorf("empty token recieved")
  }
  return token, nil
}

func (f *Fetcher) getTickersResponse(query string) (*tickersResponse, error) {
  reqUrl := fmt.Sprint(basePrefixAPI, tickersAPI, "?", query)
  resp, err := f.client.Get(reqUrl)
  if err != nil {
    return nil, fmt.Errorf("cannot get response: %v", err)
  }
  tickersResp := &tickersResponse{}
  if err = json.Unmarshal(resp, tickersResp); err != nil {
    return nil, fmt.Errorf("cannot parse json response: %v", err)
  }
  return tickersResp, nil
}

func (f *Fetcher) fetchTickers(fetchStock func(*domain.Ticker) error) error {
  query := url.Values{}
  query.Add("apiKey", f.token)
  query.Add("active", "true")
  query.Add("order", "asc")

  for {
    tickersResp, err := f.getTickersResponse(query.Encode())
    if err != nil {
      return err
    }
    respStatus := tickersResp.Status
    cursorUrl := tickersResp.NextUrl

    if respStatus != respStatusOK {
      return fmt.Errorf("bad response status: %s", respStatus)
    }
    if tickersResp.Count == 0 || cursorUrl == "" {
      break
    }

    for _, tickerResp := range tickersResp.Results {
      if tickerResp == nil {
        continue
      }
      ticker := createTicker(tickerResp)

      if err := f.tickersCache.Put(ticker); err != nil {
        log.Errorf("cannot put tocker in storage: %v", err)
      }

      if err = fetchStock(ticker); err != nil {
        log.Warnf("cannot fetch stock: %v", err)
        continue
      }
    }

    cursor, err := url.Parse(cursorUrl)
    if err != nil {
      return fmt.Errorf("cannot parse cursor url: %s", cursorUrl)
    }

    cursorValue := cursor.Query().Get(cursorKey)
    if cursorValue == "" {
      break
    }

    query.Add(cursorKey, cursorValue)
  }
  return nil
}

func createTicker(resp *tickerResult) *domain.Ticker {
  return &domain.Ticker{
    Ticker:    resp.Ticker,
    Name:      resp.Name,
    Locale:    resp.Locale,
    Active:    resp.Active,
    Currency:  resp.CurrencyName,
    UpdatedAt: resp.LastUpdatedUtc,
  }
}

func (f *Fetcher) getStockDateRange() (string, string) {
  nowT := time.Now()

  fromT := nowT
  toT := nowT

  sub := hoursPerDay
  if f.state.lastMode == modeTotal {
    sub = hoursPerYear
  }
  dur := time.Duration(sub) * time.Hour

  from := fromT.Add(-dur).Format("2006-01-02")
  to := toT.Format("2006-01-02")

  return from, to
}

func (f *Fetcher) fetchStocks(ticker *domain.Ticker) error {
  stocksTicker := ticker.Ticker
  multiplier := 1
  timespan := "day"
  from, to := f.getStockDateRange()

  rangeQuery := fmt.Sprintf(pricesAPI, stocksTicker, multiplier, timespan, from, to)

  query := url.Values{}
  query.Add("apiKey", f.token)

  reqUrl := fmt.Sprint(basePrefixAPI, rangeQuery, "?", query.Encode())

  resp, err := f.client.Get(reqUrl)
  if err != nil {
    return fmt.Errorf("cannot get response")
  }

  pricesResp := &pricesResponse{}
  if err := json.Unmarshal(resp, pricesResp); err != nil {
    return fmt.Errorf("cannot parse json reponse: %v", err)
  }

  if pricesResp.QueryCount == 0 {
    return fmt.Errorf("stock prices not found: %v", err)
  }

  for _, stockPrice := range pricesResp.StockResults {
    stock := createStock(ticker, stockPrice)

    if err := f.stocksCache.Put(stock); err != nil {
      return fmt.Errorf("cannot put stock in storage: %v", err)
    }
  }
  return nil
}

func createStock(ticker *domain.Ticker, resp *pricesResult) *domain.Stock {
  return &domain.Stock{
    Ticker:    ticker.Ticker,
    Open:      resp.Open,
    Close:     resp.Close,
    Highest:   resp.Highest,
    Lowest:    resp.Lowest,
    Timestamp: resp.Timestamp,
    Volume:    resp.Volume,
  }
}
