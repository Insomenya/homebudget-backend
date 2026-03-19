package config

import "os"

type Config struct {
	Port       string
	DBPath     string
	CORSOrigin string
}

func Load() *Config {
	return &Config{
		Port:       env("PORT", "8080"),
		DBPath:     env("DB_PATH", "homebudget.db"),
		CORSOrigin: env("CORS_ORIGIN", "*"),
	}
}

func env(key, fb string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fb
}