package main

import (
	"flag"
	"github.com/joho/godotenv"
	"log"
	"os"
	"strings"
	"sync"
)

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

type OAuth struct {
	github struct {
		clientId     string
		clientSecret string
		redirectUrl  string
		scopes       []string
	}
	google struct {
		clientId     string
		clientSecret string
		redirectUrl  string
		scopes       []string
	}
}

type GRPC struct {
	auth struct {
		host string
		port string
	}
}

type Token struct {
	secretKey                  string
	issuer                     string
	accessTokenExpirationTime  int
	refreshTokenExpirationTime int
}

type Config struct {
	db    DB
	t     Token
	grpc  GRPC
	oauth OAuth
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

		flag.StringVar(&instance.grpc.auth.host, "grpc-job-host", os.Getenv("GRPC_JOB_HOST"), "Job service host")
		flag.StringVar(&instance.grpc.auth.port, "grpc-job-port", os.Getenv("GRPC_JOB_PORT"), "Job service port")

		flag.StringVar(&instance.oauth.github.clientId, "github-client-id", os.Getenv("GITHUB_CLIENT_ID"), "Github client id")
		flag.StringVar(&instance.oauth.github.clientSecret, "github-client-secret", os.Getenv("GITHUB_CLIENT_SECRET"), "Github client secret")
		flag.StringVar(&instance.oauth.github.redirectUrl, "github-redirect-url", os.Getenv("GITHUB_REDIRECT_URL"), "Github redirect url")

		var githubScopes string
		flag.StringVar(&githubScopes, "github-scopes", os.Getenv("GITHUB_SCOPES"), "Github scopes")
		if githubScopes == "" {
			githubScopes = "read:user,user:email"
		}
		instance.oauth.github.scopes = strings.Split(githubScopes, ",")

		flag.StringVar(&instance.oauth.google.clientId, "google-client-id", os.Getenv("GOOGLE_CLIENT_ID"), "Google client id")
		flag.StringVar(&instance.oauth.google.clientSecret, "google-client-secret", os.Getenv("GOOGLE_CLIENT_SECRET"), "Google client secret")
		flag.StringVar(&instance.oauth.google.redirectUrl, "google-redirect-url", os.Getenv("GOOGLE_REDIRECT_URI"), "Google redirect url")

		var googleScopes string
		flag.StringVar(&googleScopes, "google-scopes", os.Getenv("GOOGLE_SCOPES"), "Google scopes")
		if googleScopes == "" {
			googleScopes = "openid,email,profile"
		}
		instance.oauth.google.scopes = strings.Split(googleScopes, ",")

		flag.Parse()
	})

	return instance
}
