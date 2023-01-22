package polygon

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
