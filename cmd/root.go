package cmd

import (
	"caching-proxy/server"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var ProxyOrigin string
var ProxyPort int
var clearCache bool

func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port number: %d (must be between 1 and 65535)", port)
	}
	return nil
}

func validateOrigin(rawURL string) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsed.Path != "" && parsed.Path != "/" {
		return fmt.Errorf("origin must not contain a path (got: %q)", parsed.Path)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("origin must use http or https (got: %q)", parsed.Scheme)
	}

	if parsed.Host == "" {
		return fmt.Errorf("origin must include a host")
	}

	return nil
}

func startProxyServer(){
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() 

	mux := http.NewServeMux()
	mux.HandleFunc("/", server.ProxyHandler)
	mux.HandleFunc("/shutdown", server.ShutdownHandler(cancel))
	address := fmt.Sprintf(":%v",ProxyPort)
	srv := &http.Server{
		Addr:   address,
		Handler: mux,
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigChan 
		log.Printf("Received OS signal '%v'. Initiating graceful shutdown...", sig)
		cancel()
	}()

	autoShutdownTimer := time.AfterFunc(server.AutoShutdownDuration, func() {
		log.Printf("Automatic shutdown triggered after %v. Initiating graceful shutdown...", server.AutoShutdownDuration)
		cancel() 
	})
	defer autoShutdownTimer.Stop() 

	go func() {
		log.Printf("Starting caching proxy server on :%v", ProxyPort)
		log.Printf("Proxying requests for %s", server.TargetURL)
		log.Printf("Default cache TTL: %v", server.CacheTTL)
		log.Printf("Automatic shutdown in %v if no explicit command.", server.AutoShutdownDuration)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
		log.Println("Server stopped listening for new connections.")
	}()

	<-ctx.Done()
	log.Println("Shutdown signal received. Performing graceful shutdown...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), server.ShutdownTimeout)
	defer shutdownCancel() 
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server graceful shutdown failed: %v", err)
	}
	log.Println("Server gracefully shut down.")

	server.ClearCache()
	log.Println("Proxy server exited.")


}

var rootCmd = &cobra.Command{
	Use:   "caching-proxy",
	Short: "Start a caching server on a defined port and cache the requests for a defined URL",
	Long: `Through this command, go starts a caching proxy server,
   it will forward requests to the actual server and cache the responses.
   If the same request is made again, it will return the cached response instead
    of forwarding the request to the server.
	When run without a specific subcommand:
  - Use --port and --origin together to start the proxy.
  - Use --clear-cache to perform cache clearing functionality.
  - You cannot use --clear-cache with --port or --origin.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error{
			portSet := cmd.Flags().Changed("port")
			originSet := cmd.Flags().Changed("origin")
			clearCacheSet := cmd.Flags().Changed("clear-cache")

			if clearCacheSet {
				fmt.Println("Performing cache clearing functionality...")
				fmt.Println("Cache cleared successfully!")
				return nil 
			}

			if portSet || originSet {
				if !portSet {
					return fmt.Errorf("error: --origin requires --port to also be set")
				}

				err := validatePort(ProxyPort)
				if err!=nil{
					return err

				}
				server.ProxyPort = strconv.Itoa(123)
				if !originSet {
					return fmt.Errorf("error: --port requires --origin to also be set")
				}
				err = validateOrigin(ProxyOrigin)
				if err!=nil{
					return err
				}
				server.TargetURL = ProxyOrigin
				fmt.Printf("Starting Caching Proxy on port %[1]d, targeting origin: %s\n. Please go to https://localhost:%[1]d/{add-your-path-here}\n", ProxyPort, ProxyOrigin)
				startProxyServer()

			}

			fmt.Println("No operation specified. Please provide valid flags.")
			return cmd.Help() 
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.PersistentFlags().IntVarP(&ProxyPort, "port", "p", 0, "The port to listen on for proxy connections (required with --origin to start proxy)")
	rootCmd.PersistentFlags().StringVarP(&ProxyOrigin, "origin", "o", "", "The origin server URL (required with --port to start proxy)")
	rootCmd.PersistentFlags().BoolVarP(&clearCache, "clear-cache", "c", false, "Clear the proxy cache (cannot be used with --port or --origin)")
	rootCmd.MarkFlagsMutuallyExclusive("port", "clear-cache")
	rootCmd.MarkFlagsMutuallyExclusive("origin", "clear-cache")

}
