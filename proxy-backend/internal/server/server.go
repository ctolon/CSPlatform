package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/rs/zerolog"

	"v0/internal/app/api/routes"
	"v0/internal/app/constants"
	"v0/internal/app/render"
	"v0/internal/config"
)

// Start Backend Server
func StartServer(
	config *config.AppConfig,
	log zerolog.Logger,
) {

	tmpl, err := render.ParseHtmlTemplates()
	if err != nil {
		panic(err)
	}

	// Register API And Proxy Routes
	e := routes.RegisterRoutes(log, config, tmpl)

	// Write Registered ROutes
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "METHOD\tPATH\tHANDLER")
	for _, r := range e.Routes() {
		fmt.Fprintf(w, "%s\t%s\t%s\n", r.Method, r.Path, r.Name)
	}
	w.Flush()

	// Configure Server
	s := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.AppPort),
		Handler: e,
	}
	s.TLSConfig = &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS12,
		CipherSuites:       constants.TLSCiphers,
	}

	// Run Server w/ Gracefully shutdown
	go func() {
		if config.AppWithTLS {
			log.Info().Msg("Start Server with TLS...")
			if err := s.ListenAndServeTLS(config.AppTLSCrt, config.AppTLSKey); err != nil && err != http.ErrServerClosed {
				log.Fatal().Msgf("error on starting server: %s", err)
			}
		} else {
			log.Info().Msg("Start Server without TLS...")
			if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatal().Msgf("error on starting server: %s", err)
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := s.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Msgf("Server forced to shutdown %s", err)
	}
	<-shutdownCtx.Done()

}
