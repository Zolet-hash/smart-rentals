package config

import "time"

type config struct {
	Server struct {
		Port string
		Host string
		ReadTimeout time.Duration
		WriteTimeout time.Duration
	}
	Database struct{
		Host string
		Port string
		User string
		Password string
		DBName string
		SSLMode string

	}
	JWT struct{
		Secret string
		TokenExpiry time.Duration
		RefreshExpiry time.Duration
	}
	Environment string
}

