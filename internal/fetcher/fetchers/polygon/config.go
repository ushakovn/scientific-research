package polygon

import (
  "fmt"
  "scientific-research/internal/queue/rabbitmq"
  "scientific-research/internal/storage/postgres"
  "scientific-research/pkg/utils/config"
  "scientific-research/pkg/utils/validation"
)

type Config struct {
  ModeTotalHours   int              `yaml:"total_mode_hours" required:"true"`
  ModeCurrentHours int              `yaml:"current_mode_hours" required:"true"`
  ApiToken         string           `yaml:"api_token" required:"true"`
  StorageConfig    *postgres.Config `yaml:"storage_config" required:"true"`
  QueueConfig      *rabbitmq.Config `yaml:"queue_config" required:"true"`
}

func NewConfig() *Config {
  return &Config{}
}

func (c *Config) Parse(configPath string) error {
  if c == nil {
    return fmt.Errorf("postgres config is a nil")
  }
  if err := config.ParseYamlConfig(configPath, c); err != nil {
    return fmt.Errorf("cannot parse yaml config: %v", err)
  }
  return validation.CheckRequiredFields(c)
}
