package utils

import (
  "bytes"
  "encoding/csv"
  "fmt"
  "reflect"
)

func CreateCSV(d []any) ([]byte, error) {
  var (
    res       []string
    hasHeader bool
  )
  b := &bytes.Buffer{}

  for _, dI := range d {
    wr := csv.NewWriter(b)

    val := reflect.ValueOf(dI).Elem()

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
