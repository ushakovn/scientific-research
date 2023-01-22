package utils

import (
  "encoding/json"
  "fmt"
  "reflect"
)

const mandatoryTag = "mandatory"

func unmarshalJSON(data []byte, v any) error {
  err := json.Unmarshal(data, v)
  if err != nil {
    return err
  }
  fields := reflect.ValueOf(v).Elem()
  for i := 0; i < fields.NumField(); i++ {
    tag, ok := fields.Type().Field(i).Tag.Lookup(mandatoryTag)
    if !ok {
      continue
    }
    if tag == "true" && fields.Field(i).IsZero() {
      fieldName := fields.Type().Field(i).Name
      return fmt.Errorf("field '%s' is mandatory", fieldName)
    }
  }
  return nil
}

func CheckMandatoryFields(v any) error {
  b, err := json.Marshal(v)
  if err != nil {
    return nil
  }
  cv := v
  return unmarshalJSON(b, cv)
}

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
