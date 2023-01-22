package slice

func ExtractOptional[T any](optional ...T) T {
  var value T

  for _, val := range optional {
    value = val
  }
  return value
}
