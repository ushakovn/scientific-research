package polygon

import (
  "context"
  "encoding/json"
  "fmt"
  "net/url"
  "os"
  "scientific-research/internal/domain"
  "scientific-research/internal/fetcher"
  "scientific-research/internal/httpclient"
  "scientific-research/internal/storage"
  "sync"
  "time"

  log "github.com/sirupsen/logrus"
)

type Fetcher struct {
  ctx     context.Context
  client  *httpclient.Client
  storage storage.Storage
  token   string
  state   *state
  once    *sync.Once
  ticker  *domain.Ticker
}

func NewFetcher(ctx context.Context) (fetcher.Fetcher, error) {
  token, err := getAccessToken()
  if err != nil {
    return nil, err
  }

  state, err := initState()
  if err != nil {
    return nil, err
  }

  client := httpclient.NewClient(
    httpclient.WithContext(ctx),
    httpclient.WithRequestsLimit(
      polygonReqsLimit,
      polygonReqPerDur,
      polygonWaitDur,
      polygonDeadlineDur,
    ),
  )

  fetcherStorage, err := storage.NewStorage()
  if err != nil {
    return nil, err
  }

  return &Fetcher{
    ctx:     ctx,
    client:  client,
    storage: fetcherStorage,
    token:   token,
    state:   state,
    once:    &sync.Once{},
  }, nil
}

func getAccessToken() (string, error) {
  token := os.Getenv(polygonTokenEnvName)
  if token == "" {
    return "", fmt.Errorf("invalid empty token recieved")
  }
  return token, nil
}

func (f *Fetcher) getTickersResponse(query string) (*tickersResponse, error) {
  reqURL := fmt.Sprint(basePrefixAPI, tickersAPI, "?", query)

  resp, err := f.client.Get(reqURL)
  if err != nil {
    return nil, fmt.Errorf("cannot get response: %v", err)
  }

  tickersResp := &tickersResponse{}
  if err = json.Unmarshal(resp, tickersResp); err != nil {
    return nil, fmt.Errorf("cannot parse json response: %v", err)
  }

  return tickersResp, nil
}

func buildTickersQuery(tokenValue string) url.Values {
  query := url.Values{}
  query.Add("apiKey", tokenValue)
  query.Add("active", "true")
  query.Add("order", "asc")

  return query
}

func (f *Fetcher) fetchTickers(fetchStocks func(ticker *domain.Ticker) error) error {
  query := buildTickersQuery(f.token)

  for {
    tickersResp, err := f.getTickersResponse(query.Encode())
    if err != nil {
      return err
    }

    respStatus := tickersResp.Status
    cursorURL := tickersResp.NextUrl

    if respStatus != respStatusOK {
      return fmt.Errorf("bad response status: %s", respStatus)
    }

    if tickersResp.Count == 0 || cursorURL == "" {
      break
    }

    for _, tickerResp := range tickersResp.Results {
      if tickerResp == nil {
        continue
      }
      ticker := createTicker(tickerResp)

      if err = f.storage.PutTicker(ticker); err != nil {
        return fmt.Errorf("cannot put ticker in database: %v", err)
      }

      if err = fetchStocks(ticker); err != nil {
        return fmt.Errorf("cannot fetch stocks for ticker %s: %v", ticker.Ticker, err)
      }
    }

    cursor, err := url.Parse(cursorURL)
    if err != nil {
      return fmt.Errorf("cannot parse cursor URL: %s", cursorURL)
    }

    cursorValue := cursor.Query().Get(respCursorKey)
    if cursorValue == "" {
      break
    }

    query.Add(respCursorKey, cursorValue)
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

  sub := f.state.modeCurrentHours

  if f.state.lastModeCode == fetcherModeTotal {
    sub = f.state.modeTotalHours
  }

  dur := time.Duration(sub) * time.Hour

  from := fromT.Add(-dur).Format("2006-01-02")
  to := toT.Format("2006-01-02")

  return from, to
}

func buildStocksReqURL(tickerName, fromDate, toDate, tokenValue string) string {
  multiplier := 1
  timespan := "day"

  rangeQuery := fmt.Sprintf(stocksAPI, tickerName, multiplier, timespan, fromDate, toDate)

  query := url.Values{}
  query.Add("apiKey", tokenValue)

  reqURL := fmt.Sprint(basePrefixAPI, rangeQuery, "?", query.Encode())

  return reqURL
}

func (f *Fetcher) fetchStocks(ticker *domain.Ticker) error {
  fromDate, toDate := f.getStockDateRange()

  reqURL := buildStocksReqURL(ticker.Ticker, fromDate, toDate, f.token)

  resp, err := f.client.Get(reqURL)
  if err != nil {
    return fmt.Errorf("cannot get response")
  }

  stockResp := &stocksResponse{}
  if err := json.Unmarshal(resp, stockResp); err != nil {
    return fmt.Errorf("cannot parse json reponse: %v", err)
  }

  if stockResp.QueryCount == 0 {
    log.Warnf("stock prices not found for ticker: %s", ticker.Ticker)
    return nil
  }

  for _, stockRes := range stockResp.StockResults {
    stock := createStock(ticker, stockRes)

    if err = f.storage.PutStock(stock); err != nil {
      return fmt.Errorf("cannot put stock in database: %v", err)
    }
  }

  return nil
}

func createStock(ticker *domain.Ticker, res *stockResult) *domain.Stock {
  return &domain.Stock{
    Ticker:    ticker.Ticker,
    Open:      res.Open,
    Close:     res.Close,
    Highest:   res.Highest,
    Lowest:    res.Lowest,
    Timestamp: res.Timestamp,
    Volume:    res.Volume,
  }
}
