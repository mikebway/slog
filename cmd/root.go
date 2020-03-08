/*
Package cmd implements command line parsing via the Cobra and Viper modules for the slog CLI utility.
The slog utility manages web access logs stored in S3.
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	unitTesting  = false // Set to true when running unit tests
	executeError error   // The error value obtained by Execute(), captured for unit test purposes
	cfgFile      string  // When configureed from a file, the location of the file
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "slog",
	Short: "The slog utility manages web access logs stored in S3",
	Long: `Slog is a CLI utility for reading and culling web access logs stored in S3.

Typically, the logs managed are those generated in response to access to static web assets
themselves served directly from S3.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if executeError = rootCmd.Execute(); executeError != nil {
		fmt.Println(executeError)
		if !unitTesting {
			os.Exit(1)
		}
	}
}

func init() {

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.slog)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//  rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
