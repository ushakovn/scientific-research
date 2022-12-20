package tinkoff // Package tinkoff: unused

import "time"

type Stock struct {
  Ticker             string                  `json:"ticker"`
  Name               string                  `json:"name"`
  Country            string                  `json:"country"`
  Logo               string                  `json:"logo"`
  Price__            *StockPrice             `json:"price__"`
  HistoricalPrices__ []*StockHistoricalPrice `json:"historical_prices__"`
}

type StockPrice struct {
  Ticker   string    `json:"ticker"`
  Value    float64   `json:"value" `
  Currency string    `json:"currency"`
  Time     time.Time `json:"time"`
}

type StockHistoricalPrice struct {
  Ticker string    `json:"ticker"`
  Value  float64   `json:"value"`
  Time   time.Time `json:"time"`
}

type stocksPayload struct {
  PaginationStart int    `json:"start"`
  PaginationEnd   int    `json:"end"`
  StockCountry    string `json:"country"`
  OrderType       string `json:"orderType"`
  SortType        string `json:"sortType"`
}

type stocksResponse struct {
  Payload struct {
    Total  int          `json:"total"`
    Values []*stockResp `json:"values"`
  } `json:"payload"`
  Time   time.Time `json:"time"`
  Status string    `json:"status"`
}

type stockResp struct {
  Symbol struct {
    Ticker       string    `json:"ticker"`
    Type         string    `json:"symbolType"`
    ClassCode    string    `json:"classCode"`
    Currency     string    `json:"currency"`
    StockName    string    `json:"showName"`
    Country      string    `json:"countryOfRiskBriefName"`
    Logo         string    `json:"countryOfRiskLogoUrl"`
    BrandName    string    `json:"brand"`
    SessionOpen  time.Time `json:"sessionOpen"`
    SessionClose time.Time `json:"sessionClose"`
    TimeToOpen   int       `json:"timeToOpen"`
  } `json:"symbol"`
  Prices struct {
    Buy struct {
      Currency string  `json:"currency"`
      Value    float64 `json:"value"`
    } `json:"buy"`
    Sell struct {
      Currency string  `json:"currency"`
      Value    float64 `json:"value"`
    } `json:"sell"`
    Last struct {
      Currency string  `json:"currency"`
      Value    float64 `json:"value"`
    } `json:"last"`
    Close struct {
      Currency string  `json:"currency"`
      Value    float64 `json:"value"`
    } `json:"close"`
  } `json:"prices"`
  Price struct {
    Currency string  `json:"currency"`
    Value    float64 `json:"value"`
  } `json:"price"`
  ExchangeStatus   string `json:"exchangeStatus"`
  HistoricalPrices []struct {
    Value    float64   `json:"amount"`
    Time     time.Time `json:"time"`
    UnixTime int       `json:"unixtime"`
    Profit   struct {
      Absolute struct {
        Currency string  `json:"currency"`
        Value    float64 `json:"value"`
      } `json:"absolute"`
      Relative float64 `json:"relative"`
    } `json:"earningsInfo"`
  } `json:"historicalPrices"`
}
