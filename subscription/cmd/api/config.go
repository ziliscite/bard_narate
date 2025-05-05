package main

import (
	"flag"
	"github.com/joho/godotenv"
	"log"
	"os"
	"sync"
)

type Payment struct {
	serverKey string
	url       string
}

type DB struct {
	host string
	port string
	user string
	pass string
	db   string
	ssl  bool
}

func (d DB) dsn() string {
	dsn := "postgres://" + d.user + ":" + d.pass + "@" + d.host + ":" + d.port + "/" + d.db
	if !d.ssl {
		return dsn + "?sslmode=disable"
	}
	return dsn
}

type Config struct {
	db DB
	pm Payment
}

var (
	instance Config
	once     sync.Once
)

func getConfig() Config {
	once.Do(func() {
		instance = Config{}

		if err := godotenv.Load(); err != nil {
			log.Printf("Error loading .env file")
		}

		flag.StringVar(&instance.db.host, "db-host", os.Getenv("POSTGRES_HOST"), "Database host")
		flag.StringVar(&instance.db.port, "db-port", os.Getenv("POSTGRES_PORT"), "Database port")
		flag.StringVar(&instance.db.user, "db-user", os.Getenv("POSTGRES_USER"), "Database user")
		flag.StringVar(&instance.db.pass, "db-pass", os.Getenv("POSTGRES_PASSWORD"), "Database password")
		flag.StringVar(&instance.db.db, "db-db", os.Getenv("POSTGRES_DB"), "Database name")

		var ssl bool
		sslStr := os.Getenv("POSTGRES_SSL")
		if sslStr == "true" {
			ssl = true
		} else {
			ssl = false
		}

		flag.BoolVar(&instance.db.ssl, "db-ssl", ssl, "Database ssl")

		flag.StringVar(&instance.pm.serverKey, "mt-server", os.Getenv("MIDTRANS_SERVER_KEY"), "Server Key")
		flag.StringVar(&instance.pm.url, "mt-url", os.Getenv("MIDTRANS_URL"), "URL")

		flag.Parse()
	})

	return instance
}
