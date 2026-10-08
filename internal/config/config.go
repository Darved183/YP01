package config

import(
	"os"
)

type Config struct {
	Addr string
}

func Load() Config{
	return Config {
		Addr: os.Getenv("HOST")+":"+ os.Getenv("PORT"),
	}
}