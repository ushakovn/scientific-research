package convert

import (
  "encoding/json"
  "fmt"
)

func MapFields(scr any, dst any) error {
  b, err := json.Marshal(scr)
  if err != nil {
    return fmt.Errorf("cannot convert to json: %v", err)
  }
  err = json.Unmarshal(b, dst)
  if err != nil {
    return fmt.Errorf("cannot parse json: %v", err)
  }
  return nil
}

func ToBatchCh[T any](batchSize int, ch chan T, batchCh chan []T) error {
  if batchSize <= 0 {
    return fmt.Errorf("non positive batch size: %v", batchSize)
  }
  var buff []T

  for item := range ch {
    if len(buff) <= batchSize {
      buff = append(buff, item)
      continue
    }
    batchCh <- buff
    buff = buff[:0]
  }

  return nil
}
