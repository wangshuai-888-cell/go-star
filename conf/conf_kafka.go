package conf

type Kafka struct {
	Brokers  []string `yaml:"brokers"`  // 例如 ["43.108.109.73:9092"]
	Topic    string   `yaml:"topic"`    // 默认主题
	Group    string   `yaml:"group"`    // 消费组（本地开发可用 go-star-local）
	Username string   `yaml:"username"` // 无 SASL 则留空
	Password string   `yaml:"password"`
}
