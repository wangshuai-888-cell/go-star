package conf

type Kafka struct {
	Brokers      []string `yaml:"brokers"`       // 例如 ["43.108.109.73:9092"]
	Topic        string   `yaml:"topic"`         // 默认主题
	Group        string   `yaml:"group"`         // 消费组（本地开发可用 go-star-local）
	MessageTopic string   `yaml:"message_topic"` // 站内信：go-star-message
	MessageGroup string   `yaml:"message_group"` // 站内信消费组
	Username     string   `yaml:"username"`      // 无 SASL 则留空
	Password     string   `yaml:"password"`
	BloomTopic   string   `yaml:"bloom_topic"` // 布隆广播：go-start-bloom
	BloomGroup   string   `yaml:"bloom_group"` // 布隆广播组前缀（实际 group = 前缀 + 实例标识）
}
