package httpclient

import (
  "net/http"
  "context"
  "fmt"
  "scientific-research/pkg/utils"
  "io"
  "strings"
  "bytes"
  "golang.org/x/sync/semaphore"
)

const (
  limiterCount = 100
)

type Client struct {
  ctx     context.Context
  client  http.Client
  limiter *semaphore.Weighted
}

type Options func(c *Client)

func NewClient(ctx context.Context, options ...Options) *Client {
  client := &Client{
    ctx:     ctx,
    client:  http.Client{},
    limiter: semaphore.NewWeighted(limiterCount),
  }
  for _, opt := range options {
    opt(client)
  }
  return client
}

func WithLimiter(count int) Options {
  return func(c *Client) {
    c.limiter = semaphore.NewWeighted(int64(count))
  }
}

type Header map[string]string

func extractOptionalHeaders(headers []Header) Header {
  var header Header
  for _, h := range headers {
    if h == nil {
      continue
    }
    header = h
  }
  return header
}

func convertToHttpHeaders(headers Header) http.Header {
  httpHeaders := http.Header{}
  for key, value := range headers {
    if key == "" {
      continue
    }
    httpHeaders.Add(key, value)
  }
  return httpHeaders
}

func (c *Client) Get(requestURL string, headers ...Header) ([]byte, error) {
  var (
    resp *http.Response
    err  error
  )
  err = utils.DoWithRetry(func() error {
    resp, err = c.getOnce(requestURL, headers)
    return err
  })
  if err != nil {
    return nil, urlError(requestURL, fmt.Sprintf("get request failed: %v", err))
  }
  if err != nil {
    return nil, urlError(requestURL, fmt.Sprintf("get request failed: %v", err))
  }
  content, err := readResponse(requestURL, resp)
  if err != nil {
    return nil, err
  }
  return content, nil
}

func (c *Client) getOnce(requestURL string, headers []Header) (*http.Response, error) {
  err := c.limiter.Acquire(c.ctx, 1)
  if err != nil {
    return nil, fmt.Errorf("cannot acquire limiter: %v", err)
  }
  defer c.limiter.Release(1)
  req, err := http.NewRequestWithContext(c.ctx, http.MethodGet, requestURL, nil)
  if err != nil {
    return nil, urlError(requestURL, fmt.Sprintf("cannot create request: %v", err))
  }
  resp, err := c.doRequest(req, headers)
  if err != nil {
    return nil, err
  }
  return resp, nil
}

func (c *Client) doRequest(req *http.Request, headers []Header) (*http.Response, error) {
  req.Header = convertToHttpHeaders(extractOptionalHeaders(headers))
  resp, err := c.client.Do(req)
  if err != nil {
    return nil, urlError(req.URL.String(), fmt.Sprintf("cannot get response: %v", err))
  }
  statusCode := resp.StatusCode
  if statusCode >= http.StatusBadRequest {
    return nil, urlError(req.URL.String(), fmt.Sprintf("bad response, got status code: %d", statusCode))
  }
  return resp, nil
}

func readResponse(requestURL string, resp any) ([]byte, error) {
  typedRes, castOk := resp.(*http.Response)
  if !castOk {
    return nil, urlError(requestURL, fmt.Sprintf("bad response"))
  }
  content, err := io.ReadAll(typedRes.Body)
  if err != nil {
    return nil, urlError(requestURL, fmt.Sprintf("cannot read response: %v", err))
  }
  if err = typedRes.Body.Close(); err != nil {
    return nil, urlError(requestURL, fmt.Sprintf("cannot close response reader: %v", err))
  }
  return content, nil
}

func preparePostPayload(requestURL string, payload any) (io.Reader, error) {
  var reader io.Reader
  switch t := payload.(type) {
  case string:
    reader = strings.NewReader(payload.(string))
  case []byte:
    reader = bytes.NewBuffer(payload.([]byte))
  default:
    return nil, urlError(requestURL, fmt.Sprintf("unsupported payload type: %T", t))
  }
  return reader, nil
}

func (c *Client) Post(requestURL string, payload any, headers ...Header) ([]byte, error) {
  var (
    resp *http.Response
    err  error
  )
  err = utils.DoWithRetry(func() error {
    resp, err = c.postOnce(requestURL, payload, headers)
    return err
  })
  if err != nil {
    return nil, urlError(requestURL, fmt.Sprintf("post request failed: %v", err))
  }
  content, err := readResponse(requestURL, resp)
  if err != nil {
    return nil, err
  }
  return content, nil
}

func (c *Client) postOnce(requestUrl string, payload any, headers []Header) (*http.Response, error) {
  err := c.limiter.Acquire(c.ctx, 1)
  defer c.limiter.Release(1)
  if err != nil {
    return nil, fmt.Errorf("cannot acquire limiter: %v", err)
  }
  reader, err := preparePostPayload(requestUrl, payload)
  if err != nil {
    return nil, urlError(requestUrl, fmt.Sprintf("cannot prepare post payload: %v", err))
  }
  req, err := http.NewRequestWithContext(c.ctx, http.MethodPost, requestUrl, reader)
  if err != nil {
    return nil, urlError(requestUrl, fmt.Sprintf("cannot create request: %v", err))
  }
  resp, err := c.doRequest(req, headers)
  if err != nil {
    return nil, err
  }
  return resp, err
}

func urlError(requestURL string, msg string) error {
  return fmt.Errorf("url: %v. %s", requestURL, msg)
}
