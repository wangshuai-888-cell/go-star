package conf

type Jwt struct {
	Expire        int    `yaml:"expire"`         // access 过期：小时
	RefreshExpire int    `yaml:"refresh_expire"` // refresh 过期：小时，例如 168=7天
	Secret        string `yaml:"secret"`
	Issuer        string `yaml:"issuer"`
}
