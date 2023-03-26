package rabbitmq

import (
  "context"
  "fmt"
  "scientific-research/pkg/utils/retries"

  ampq "github.com/rabbitmq/amqp091-go"
)

type Client interface {
  QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args ampq.Table) (ampq.Queue, error)
  PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg ampq.Publishing) error
  Consume(queue, cons string, autoAck, excl, noLocal, noWait bool, args ampq.Table) (<-chan ampq.Delivery, error)
}

func NewClient(config *Config) (Client, error) {
  strConn := config.ConnectString()

  var (
    conn *ampq.Connection
    ch   *ampq.Channel
    err  error
  )
  err = retries.DoWithRetry(func() error {
    if conn, err = ampq.Dial(strConn); err != nil {
      return fmt.Errorf("cannot connect to rabbitmq: %v", err)
    }
    return nil
  })
  if err != nil {
    return nil, fmt.Errorf("connection to rabbitmq failed: %v", err)
  }

  err = retries.DoWithRetry(func() error {
    if ch, err = conn.Channel(); err != nil {
      return fmt.Errorf("cannot open ampq server channel: %v", err)
    }
    return nil
  })
  if err != nil {
    return nil, fmt.Errorf("ampq server channel opening failed: %v", err)
  }

  return ch, nil
}
