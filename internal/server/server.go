package server

import (
	"context"
	"crypto/tls"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/go-chi/chi/v5"
	cmw "github.com/go-chi/chi/v5/middleware"
	"github.com/heyztb/lists-backend/internal/api"
	"github.com/heyztb/lists-backend/internal/cache"
	"github.com/heyztb/lists-backend/internal/database"
	"github.com/heyztb/lists-backend/internal/html"
	"github.com/heyztb/lists-backend/internal/html/static"
	"github.com/heyztb/lists-backend/internal/log"
	"github.com/heyztb/lists-backend/internal/middleware"
	security "github.com/heyztb/lists-backend/internal/paseto"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/volatiletech/sqlboiler/v4/drivers/sqlboiler-psql/driver"
)

type Config struct {
	// HTTP Server configuration
	ListenAddress string        `config:"LISTEN_ADDRESS"`
	ReadTimeout   time.Duration `config:"READ_TIMEOUT"`
	WriteTimeout  time.Duration `config:"WRITE_TIMEOUT"`
	IdleTimeout   time.Duration `config:"IDLE_TIMEOUT"`
	DisableTLS    bool          `config:"DISABLE_TLS"`
	TLSCertFile   string        `config:"TLS_CERT_FILE"`
	TLSKeyFile    string        `config:"TLS_KEY_FILE"`
	PasetoKey     string        `config:"PASETO_KEY"`
	LogFilePath   string        `config:"LOG_FILE_PATH"`

	// Backing services configuration
	DatabaseHost     string `config:"DATABASE_HOST"`
	DatabasePort     int    `config:"DATABASE_PORT"`
	DatabaseUser     string `config:"DATABASE_USER"`
	DatabasePassword string `config:"DATABASE_PASSWORD"`
	DatabaseName     string `config:"DATABASE_NAME"`
	DatabaseSSLMode  string `config:"DATABASE_SSL_MODE"`
	RedisHost        string `config:"REDIS_HOST"`
}

func Run(cfg *Config) {
	var err error
	dsn := driver.PSQLBuildQueryString(
		cfg.DatabaseUser,
		cfg.DatabasePassword,
		cfg.DatabaseName,
		cfg.DatabaseHost,
		cfg.DatabasePort,
		cfg.DatabaseSSLMode,
	)
	database.DB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	cache.Redis = redis.NewClient(&redis.Options{
		Addr: cfg.RedisHost,
		DB:   0,
	})

	if cfg.PasetoKey != "" {
		security.ServerSigningKey, err = paseto.NewV4AsymmetricSecretKeyFromHex(cfg.PasetoKey)
		if err != nil {
			log.Fatal().Err(err).Msg("faield to read paseto key")
		}
	}

	server := &http.Server{
		Addr:    cfg.ListenAddress,
		Handler: &middleware.Size{Mux: service()},
		TLSConfig: &tls.Config{
			MinVersion:               tls.VersionTLS13,
			PreferServerCipherSuites: true,
		},
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// Server run context
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	// Listen for syscall signals for process to interrupt/quit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sig

		// Shutdown signal with grace period of 30 seconds
		shutdownCtx, shutdownStopCtx := context.WithTimeout(serverCtx, 30*time.Second)

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				log.Fatal().Msg("graceful shutdown timed out, forcing exit")
			}
			shutdownStopCtx()
		}()

		// Trigger graceful shutdown
		err := server.Shutdown(shutdownCtx)
		if err != nil {
			log.Fatal().Err(err).Msg("error shutting down server")
		}
		log.Info().Msg("server shutting down")
		serverStopCtx()
	}()

	if cfg.DisableTLS {
		log.Info().Msgf("starting http server on %s", cfg.ListenAddress)
		server.ListenAndServe()
	} else {
		log.Info().Msgf("starting https server on %s", cfg.ListenAddress)
		server.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
	}

	// Wait for server context to be stopped
	<-serverCtx.Done()
	os.Exit(0)
}

func service() http.Handler {
	r := chi.NewRouter()
	r.Use(cmw.RequestID)
	r.Use(middleware.Logger)
	r.Use(cmw.Recoverer)
	static.Mount(r)

	r.Get(`/`, html.ServeHomePage)
	r.Get(`/register`, html.ServeRegisterPage)
	r.Get(`/login`, html.ServeLoginPage)
	r.Get(`/about`, html.ServeAboutPage)

	r.Get(`/api/`, api.HealthcheckHandler)
	r.Post(`/api/auth/register`, api.RegisterHandler)
	r.Post(`/api/auth/identify`, api.IdentityHandler)
	r.Post(`/api/auth/login`, api.LoginHandler)

	r.Group(func(r chi.Router) {
		r.Use(middleware.Authentication)

		r.Get(`/api/lists`, api.GetListsHandler)
		r.Get(`/api/lists/{list}`, api.GetListHandler)
		r.Delete(`/api/lists/{list}`, api.DeleteListHandler)
		r.Get(`/api/sections`, api.GetSectionsHandler)
		r.Get(`/api/sections/{section}`, api.GetSectionHandler)
		r.Delete(`/api/sections/{section}`, api.DeleteSectionHander)
		r.Get(`/api/items`, api.GetItemsHandler)
		r.Get(`/api/items/{item}`, api.GetItemHandler)
		r.Post(`/api/items/{item}/close`, api.CloseItemHandler)
		r.Post(`/api/items/{item}/reopen`, api.ReopenItemHandler)
		r.Delete(`/api/items/{item}`, api.DeleteItemHandler)
		r.Get(`/api/comments`, api.GetCommentsHandler)
		r.Get(`/api/comments/{comment}`, api.GetCommentHandler)
		r.Delete(`/api/comments/{comment}`, api.DeleteCommentHandler)
		r.Get(`/api/labels`, api.GetLabelsHandler)
		r.Get(`/api/labels/{label}`, api.GetLabelHandler)
		r.Delete(`/api/labels/{label}`, api.DeleteLabelHandler)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Decryption)
			r.Post(`/api/lists`, api.CreateListHandler)
			r.Post(`/api/lists/{list}`, api.UpdateListHandler)
			r.Post(`/api/sections`, api.CreateSectionHandler)
			r.Post(`/api/sections/{section}`, api.UpdateSectionHandler)
			r.Post(`/api/items`, api.CreateItemHandler)
			r.Post(`/api/items/{item}`, api.UpdateItemHandler)
			r.Post(`/api/comments`, api.CreateCommentHandler)
			r.Post(`/api/comments/{comment}`, api.UpdateCommentHandler)
			r.Post(`/api/labels`, api.CreateLabelHandler)
			r.Post(`/api/labels/{label}`, api.UpdateLabelHandler)
		})
	})

	return r
}
