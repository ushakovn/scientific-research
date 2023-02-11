package convert

import (
  "bytes"
  "encoding/csv"
  "encoding/json"
  "fmt"
  "reflect"
)

func ToJsonBytes[T any](s T) ([]byte, error) {
  b, err := json.Marshal(s)
  if err != nil {
    return nil, fmt.Errorf("cannot marshal to json: %v", err)
  }
  return b, nil
}

func ToCsvBytes[T any](s []T) ([]byte, error) {
  var (
    res       []string
    hasHeader bool
  )
  b := &bytes.Buffer{}

  for _, item := range s {
    wr := csv.NewWriter(b)

    val := reflect.ValueOf(item).Elem()

    if !hasHeader {
      for i := 0; i < val.NumField(); i++ {
        tagVal, ok := val.Type().Field(i).Tag.Lookup("json")
        if !ok {
          return nil, fmt.Errorf("data tag not found in domain")
        }
        res = append(res, tagVal)
      }
      if err := wr.Write(res); err != nil {
        return nil, fmt.Errorf("error writing record to file: %v", err)
      }
      hasHeader = true
      res = res[:0]
    }

    for i := 0; i < val.NumField(); i++ {
      valField := val.Field(i).Interface()

      res = append(res, fmt.Sprintf("%v", valField))
    }
    if err := wr.Write(res); err != nil {
      return nil, fmt.Errorf("error writing record to file: %v", err)
    }
    res = res[:0]

    wr.Flush()
  }
  return b.Bytes(), nil
}
