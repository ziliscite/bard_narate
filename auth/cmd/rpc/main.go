package main

import (
	"context"
	"fmt"
	"github.com/ziliscite/bard_narate/auth/internal/repository"
	"github.com/ziliscite/bard_narate/auth/internal/service"
	"github.com/ziliscite/bard_narate/auth/pkg/github"
	"github.com/ziliscite/bard_narate/auth/pkg/google"
	"github.com/ziliscite/bard_narate/auth/pkg/postgres"
	pb "github.com/ziliscite/bard_narate/auth/pkg/protobuf"
	"github.com/ziliscite/bard_narate/auth/pkg/token"
	"google.golang.org/grpc"
	"log/slog"
	"net"
	"os"
	"time"
)

func main() {
	cfg := getConfig()

	// Initialize OAuth2 configurations
	ghcfg := github.NewOAuthClient(cfg.oauth.github.clientId, cfg.oauth.github.clientSecret, cfg.oauth.github.redirectUrl, cfg.oauth.github.scopes...)
	ggcfg := google.NewOAuthClient(cfg.oauth.google.clientId, cfg.oauth.google.clientSecret, cfg.oauth.google.redirectUrl, cfg.oauth.google.scopes...)

	// Create bg context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get database pool
	pool, err := postgres.Open(ctx, cfg.db.dsn())
	if err != nil {
		panic(err)
	}

	// Migrate
	if err = postgres.AutoMigrate(cfg.db.dsn()); err != nil {
		panic(err)
	}

	// Create the repositories
	ghprov := repository.NewGitHub(ghcfg)
	ggprov := repository.NewGoogle(ggcfg)

	tkr, err := repository.NewTokenRepository(pool)
	if err != nil {
		panic(err)
	}

	ur, err := repository.NewUserRepository(pool)
	if err != nil {
		panic(err)
	}

	tm := token.NewMaker(cfg.t.secretKey, cfg.t.issuer)

	// Create the services
	as := service.NewAuthenticator(tkr, ur, tm)
	oas := service.NewOAuthAuthenticator(ghprov, ggprov, tkr, ur, tm)

	// Create the grpc controllers
	server := grpc.NewServer()

	authService := NewAuthenticationService(as, oas)

	pb.RegisterOAuthServiceServer(server, authService)
	pb.RegisterServerAuthServiceServer(server, authService)

	// Start grpc server
	listen, err := net.Listen("tcp", fmt.Sprintf("%v:%v", cfg.grpc.auth.host, cfg.grpc.auth.port))
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	defer listen.Close()

	if err = server.Serve(listen); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

/*
mux := http.NewServeMux()
mux.HandleFunc("GET /oauth2/github", func(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if state == "" {
		state = uuid.NewString()
	}

	http.Redirect(w, r, githubCfg.AuthURL(state), http.StatusTemporaryRedirect)
})

mux.HandleFunc("GET /oauth2/github/callback", func(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	token, err := githubCfg.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user, err := githubCfg.User(r.Context(), token)
	if err != nil {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
})

mux.HandleFunc("GET /oauth2/google", func(w http.ResponseWriter, r *http.Request) {
	var state string
	if state = r.URL.Query().Get("state"); state == "" {
		state = uuid.NewString()
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, googleCfg.AuthURL(state), http.StatusTemporaryRedirect)
})

mux.HandleFunc("GET /oauth2/google/callback", func(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		http.Error(w, "state cookie not found", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" || state != stateCookie.Value {
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}

	// Clear the state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	code := r.URL.Query().Get("code")
	token, err := googleCfg.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user, err := googleCfg.User(r.Context(), token)
	if err != nil {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
})

log.Printf("Server starting on port %d", cfg.port)
if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.port), mux); err != nil {
	log.Fatal(err)
}
*/
