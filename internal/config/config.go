package config

type Config struct {
	Environment Environment
	Log         Log
	HTTP        HTTPServer
	BaseURL     string `env:"BASE_URL"`
	DatabaseURL string `env:"DATABASE_URL"`

	Paypal    Paypal    `envPrefix:"PAYPAL_"`
	BrainTree Braintree `envPrefix:"BRAINTREE_"`
}

type Paypal struct {
	BaseApiURL   string `env:"BASE_API_URL"`
	ClientID     string `env:"CLIENT_ID"`
	ClientSecret string `env:"CLIENT_SECRET"`
	WebhookID    string `env:"WEBHOOK_ID"`
	RedirectURL  string `env:"REDIRECT_URL"`
}

type Braintree struct {
	Environment string `env:"ENVIRONMENT"`
	MerchantID  string `env:"MERCHANT_ID"`
	PublicKey   string `env:"PUBLIC_KEY"`
	PrivateKey  string `env:"PRIVATE_KEY"`
}

type Environment struct {
	Name string `env:"ENVIRONMENT" envDefault:"development"`
}

type Log struct {
	Level  string `env:"LOG_LEVEL" envDefault:"info"`
	Format string `env:"LOG_FORMAT" envDefault:"json"`
}

type HTTPServer struct {
	Host string `env:"HTTP_HOST" envDefault:"0.0.0.0"`
	Port string `env:"HTTP_PORT" envDefault:"8080"`
}
