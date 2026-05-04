package setup

type Config struct {
	Kind                      string
	TargetInput               string
	TargetPath                string
	PackageManager            string
	UserIDType                string
	HMACSecret                string
	DatabaseURL               string
	RedisURL                  string
	LemonSqueezyAPIKey        string
	LemonSqueezyStoreID       string
	LemonSqueezyVariantID     string
	LemonSqueezyWebhookSecret string
}

type Result struct {
	TargetPath string
	APIKey     string
	APIKeyName string
	UsedPM     string
}
