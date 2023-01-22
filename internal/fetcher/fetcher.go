package fetcher

import "scientific-research/internal/domain"

type Fetcher interface {
  ContinuouslyFetch()
  SetTicker(ticker string)
  QueryFetchedStocks(tickerName string) ([]*domain.Stock, error)
}
