package config

import "github.com/spf13/viper"

type Config struct {
	ServiceName              string `mapstructure:"SERVICE_NAME"`
	OTELExporterOTLPEndpoint string `mapstructure:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	OTELExporterOTLPHeaders  string `mapstructure:"OTEL_EXPORTER_OTLP_HEADERS"`
	OTELResourceAttreibutes  string `mapstructure:"OTEL_RESOURCE_ATTRIBUTES"`
	DBSource                 string `mapstructure:"DB_SOURCE"`
	ClerKKey                 string `mapstructure:"CLERK_KEY"`
	LogFilePath              string `mapstructure:"LOG_FILE_PATH"`
	LokiURL                  string `mapstructure:"LOKI_URL"`
	SyslogAddress            string `mapstructure:"SYSLOG_ADDRESS"`
	SyslogNetwork            string `mapstructure:"SYSLOG_NETWORK"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}
	viper.Unmarshal(&config)
	return config, nil
}
