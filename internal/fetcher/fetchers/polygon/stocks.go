package polygon

import (
  "context"
  "fmt"
  "net/url"
  "scientific-research/internal/domain"
  "scientific-research/internal/fetcher"
  "scientific-research/internal/httpclient"
  "scientific-research/internal/storage"
  "scientific-research/pkg/utils/common"
  "scientific-research/pkg/utils/timeutils"
  "scientific-research/pkg/utils/validation"
  "sync"
  "time"

  log "github.com/sirupsen/logrus"
)

type Fetcher struct {
  ctx          context.Context
  client       *httpclient.Client
  storage      storage.Storage
  token        string
  state        *state
  once         *sync.Once
  specTickerId string
}

func NewFetcher(ctx context.Context, config *Config) (fetcher.Fetcher, error) {
  client := httpclient.NewClient(
    httpclient.WithContext(ctx),
    httpclient.WithRequestsLimit(
      polygonReqsLimit,
      polygonReqPerDur,
      polygonWaitDur,
      polygonDeadlineDur,
    ),
  )
  fetcherStorage, err := storage.NewStorage(ctx, config.StorageConfig)
  if err != nil {
    return nil, err
  }
  return &Fetcher{
    ctx:     ctx,
    client:  client,
    storage: fetcherStorage,
    token:   config.AccessToken,
    state:   newFetcherState(config.ModeTotalHours, config.ModeCurrentHours),
    once:    &sync.Once{},
  }, nil
}

func (f *Fetcher) getTickersResponse(query string) (*tickersResponse, error) {
  reqURL := getReqUrlFromStateOrNew(f.state.ticker,
    func() string {
      return fmt.Sprint(basePrefixAPI, tickersAPI, "?", query)
    })

  resp, err := f.client.Get(reqURL)
  if err != nil {
    return nil, fmt.Errorf("cannot get response: %v", err)
  }
  tickersResp := &tickersResponse{}
  if err = f.client.ParseResponse(resp, tickersResp); err != nil {
    return nil, fmt.Errorf("cannot parse response: %v", err)
  }

  return tickersResp, nil
}

func buildTickersQuery(token string) url.Values {
  query := newQueryToAPI(token)
  query.Add("active", "true")
  query.Add("order", "asc")
  return query
}

func (f *Fetcher) fetchTickerDetails(tickerId string) (*domain.TickerDetails, error) {
  resp, err := f.getTickerDetailsResponse(tickerId)
  if err != nil {
    return nil, fmt.Errorf("cannot get ticker details response: %v", err)
  }
  if resp.Status != respStatusOK {
    return nil, fmt.Errorf("bad response status: %s", resp.Status)
  }
  if resp.Results == nil {
    return nil, fmt.Errorf("ticker details results not found")
  }
  details, err := createTickerDetails(resp.Results)
  if err != nil {
    return nil, fmt.Errorf("cannot create ticker details: %v", err)
  }
  return details, nil
}

func (f *Fetcher) getTickerDetailsResponse(tickerId string) (*tickerDetailsResponse, error) {
  reqURL := getReqUrlFromStateOrNew(f.state.tickerDetails,
    func() string {
      tickerDetailsQuery := fmt.Sprintf(tickerDetailsAPI, tickerId)
      return fmt.Sprint(basePrefixAPI, tickerDetailsQuery, "?", newQueryToAPI(f.token).Encode())
    })

  resp, err := f.client.Get(reqURL)
  if err != nil {
    return nil, fmt.Errorf("cannot get response: %v", err)
  }
  tickerDetailsResp := &tickerDetailsResponse{}
  if err = f.client.ParseResponse(resp, tickerDetailsResp); err != nil {
    return nil, fmt.Errorf("cannot parse response: %v", err)
  }

  return tickerDetailsResp, nil
}

func (f *Fetcher) fetchTickers() error {
  query := buildTickersQuery(f.token)
  queryStr := query.Encode()

  for {
    tickersResp, err := f.getTickersResponse(queryStr)
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
    for _, tickerRespResult := range tickersResp.Results {
      if tickerRespResult == nil {
        continue
      }
      ticker, err := createTicker(tickerRespResult)
      if err != nil {
        return fmt.Errorf("cannot create ticker: %v", err)
      }
      if err = f.storage.PutTicker(ticker); err != nil {
        return fmt.Errorf("cannot put ticker to storage: %v", err)
      }

      tickerDetails, err := f.fetchTickerDetails(ticker.TickerId)
      if err != nil {
        return fmt.Errorf("cannot fetch ticker details for ticker %s: %v", ticker.TickerId, err)
      }
      if err = f.storage.PutTickerDetails(tickerDetails); err != nil {
        return fmt.Errorf("cannot put ticker details to storage")
      }

      if err = f.fetchStocks(ticker.TickerId); err != nil {
        return fmt.Errorf("cannot fetch stocks for ticker %s: %v", ticker.TickerId, err)
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

func (f *Fetcher) getStockDateRange() (string, string) {
  nowT := time.Now()
  fromT := nowT
  toT := nowT
  sub := f.state.modeCurrentHours

  if f.state.modeCode == fetcherModeTotal {
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
  reqURL := fmt.Sprint(basePrefixAPI, rangeQuery, "?", newQueryToAPI(tokenValue).Encode())
  return reqURL
}

func newQueryToAPI(tokenValue string) url.Values {
  const tokenQueryKey = "apiKey"
  query := url.Values{}
  query.Add(tokenQueryKey, tokenValue)
  return query
}

func (f *Fetcher) fetchStocks(tickerId string) error {
  reqURL := getReqUrlFromStateOrNew(f.state.stocks,
    func() string {
      fromDate, toDate := f.getStockDateRange()
      return buildStocksReqURL(tickerId, fromDate, toDate, f.token)
    })

  resp, err := f.client.Get(reqURL)
  if err != nil {
    return fmt.Errorf("cannot get response")
  }

  stockResp := &stocksResponse{}
  if err := f.client.ParseResponse(resp, stockResp); err != nil {
    return fmt.Errorf("cannot parse reponse: %v", err)
  }

  if stockResp.QueryCount == 0 {
    log.Warnf("stock prices not found for ticker: %s", tickerId)
    return nil
  }

  for _, stockRes := range stockResp.StockResults {
    stock, err := createStock(tickerId, stockRes)
    if err != nil {
      return fmt.Errorf("cannot create stock: %v", err)
    }
    if err = f.storage.PutStock(stock); err != nil {
      return fmt.Errorf("cannot put stock to storage: %v", err)
    }
  }

  return nil
}

func createTicker(res *tickerResult) (*domain.Ticker, error) {
  if res == nil {
    return nil, nil
  }
  ticker := &domain.Ticker{
    TickerId:          res.Ticker,
    CompanyName:       res.Name,
    CompanyLocale:     res.Locale,
    CurrencyName:      res.CurrencyName,
    TickerCik:         res.Cik,
    Active:            res.Active,
    CreatedAt:         timeutils.NotTimeUTC(),
    ExternalUpdatedAt: timeutils.TimeToUTC(res.LastUpdatedUtc),
  }
  if err := validation.SetDefaultStringValues(ticker, defaultStringValue); err != nil {
    return nil, err
  }
  return ticker, nil
}

func createTickerDetails(res *tickerDetailsResults) (*domain.TickerDetails, error) {
  if res == nil {
    return nil, nil
  }
  details := &domain.TickerDetails{
    TickerId:           res.Ticker,
    CompanyDescription: res.Description,
    HomepageUrl:        res.HomepageUrl,
    PhoneNumber:        res.PhoneNumber,
    TotalEmployees:     res.TotalEmployees,
  }
  if res.Address != nil {
    details.CompanyState = res.Address.State
    details.CompanyCity = common.TitleString(res.Address.City)
    details.CompanyAddress = common.TitleString(res.Address.Address1)
    details.CompanyPostalCode = res.Address.PostalCode
    details.CreatedAt = timeutils.NotTimeUTC()
  }
  if err := validation.SetDefaultStringValues(details, defaultStringValue); err != nil {
    return nil, err
  }
  return details, nil
}

func createStock(tickerId string, res *stockResult) (*domain.Stock, error) {
  if res == nil {
    return nil, nil
  }
  const sepId = "-"
  stock := &domain.Stock{
    StockId:       fmt.Sprint(tickerId, sepId, res.Timestamp),
    TickerId:      tickerId,
    OpenPrice:     res.Open,
    ClosePrice:    res.Close,
    HighestPrice:  res.Highest,
    LowestPrice:   res.Lowest,
    TradingVolume: res.Volume,
    StockedAt:     timeutils.TimestampToTimeUTC(res.Timestamp),
    CreatedAt:     timeutils.NotTimeUTC(),
  }
  if err := validation.SetDefaultStringValues(stock, defaultStringValue); err != nil {
    return nil, err
  }
  return stock, nil
}

func getReqUrlFromStateOrNew(stateReq *stateReq, newReqURL func() string) string {
  var reqURL string
  // if request URL not used and set in state
  if !stateReq.used && stateReq.reqURL != "" {
    reqURL = stateReq.reqURL
    // use it once
    stateReq.used = true
  } else {
    // else form new request URL
    reqURL = newReqURL()
    // save it in state
    stateReq.reqURL = reqURL
  }
  return reqURL
}
