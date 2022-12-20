package fetcher

import "scientific-research/internal/domain"

type Fetcher interface {
  ContinuouslyFetch()
  SetTicker(ticker string)
  QueryFetchedStocks(ticker string) ([]*domain.Stock, error)
}
