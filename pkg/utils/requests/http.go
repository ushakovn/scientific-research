package requests

import (
  "net/http"
  "net/url"
)

type (
  ResponseHeaders map[string]string
  RequestQuery    map[string]string
)

func PlainRequestQuery(query url.Values) RequestQuery {
  plain := make(map[string]string, len(query))

  for key, values := range query {
    if len(values) == 0 {
      continue
    }
    plain[key] = values[0]
  }
  return plain
}

func ApplyResponseHeaders(w http.ResponseWriter, headers ResponseHeaders) {
  for key, value := range headers {
    w.Header().Add(key, value)
  }
}
