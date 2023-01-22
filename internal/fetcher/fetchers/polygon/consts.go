package polygon

import "time"

// Fetcher constants

const (
  fetcherModeTotal   = 0
  fetcherModeCurrent = 1
)

const (
  fetcherRetryCount  = 5
  fetcherChanBufSize = 25
)

const (
  recentlyFetchedSleepInterval  = 1 * time.Hour
  encounteredErrorSleepInterval = 10 * time.Minute
  relevantThresholdInterval     = 24 * time.Hour
)

// Polygon API constants

const (
  polygonReqsLimit   = 5
  polygonReqPerDur   = 1 * time.Minute
  polygonWaitDur     = 10 * time.Second
  polygonDeadlineDur = 120 * time.Second
)

const (
  respCursorKey = "cursor"
  respStatusOK  = "OK"
)

const (
  basePrefixAPI = "https://api.polygon.io"
  tickersAPI    = "/v3/reference/tickers"
  stocksAPI     = "/v2/aggs/ticker/%s/range/%d/%s/%s/%s"
)

const (
  polygonTokenEnvName = "POLYGON_TOKEN"
  modeTotalHoursEnv   = "MODE_TOTAL_HOURS"
  modeCurrentHoursEnv = "MODE_CUR_HOURS"
)
