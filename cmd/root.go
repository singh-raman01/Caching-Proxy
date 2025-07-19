package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

//adsas

var URLPath string
var Port int

var rootCmd = &cobra.Command{
	Use:   "caching-proxy",
	Short: "Start a caching server on a defined port and cache the requests for a defined URL",
	Long: `Through this command, go starts a caching proxy server,
   it will forward requests to the actual server and cache the responses.
   If the same request is made again, it will return the cached response instead
    of forwarding the request to the server.`,
	Args: cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) > 0 {
			fmt.Printf("Caching Proxy also received direct inputs: %s\n", strings.Join(args, " "))
		}

		fmt.Printf("The provided host is :%v, and the provided URL is :%v\n",Port,URLPath)

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
	// --- Add the two required flags to the ROOT command ---
	rootCmd.PersistentFlags().StringVarP(&URLPath, "origin", "O", "", "The host address whose data we want to cache")
	rootCmd.PersistentFlags().IntVarP(&Port, "port", "p", 0, "The port where we start the cached server")
	rootCmd.MarkPersistentFlagRequired("origin")
	rootCmd.MarkPersistentFlagRequired("port")
}
