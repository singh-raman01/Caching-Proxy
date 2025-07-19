package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

//adsas

var ProxyOrigin string
var ProxyPort int
var clearCache bool

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
				if !originSet {
					return fmt.Errorf("error: --port requires --origin to also be set")
				}
				fmt.Printf("Starting Caching Proxy on port %d, targeting origin: %s\n", ProxyPort, ProxyOrigin)
				return nil
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
