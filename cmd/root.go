package cmd

/*
Copyright © 2019 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/masahide/rcp/pkg/rcp"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var (
	cfgFile = ""
	// Rcp configs
	dummyInputString string
	r                = &rcp.Rcp{
		MaxBufNum:    100,
		BufSize:      10 * 1024 * 1024, // 10MByte
		SingleThread: false,
		DummyInput:   0,
		DummyOutput:  false,
		DialAddr:     "",
		Output:       "",
		Input:        "",
		ListenAddr:   "0.0.0.0:1987",
	}
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "rcp",
	Short: "Command for file transfer by tcp",
	Long: `Command for file transfer by tcp.

Characteristic:
- Transfer files using buffer between file read / write and transfer process
- Monitors read / write speed and transfer speed every second and displays them in sparkline chart
- Network and storage performance can be measured with dummy data send and dummy receive functions

The main procedure is performed in two steps:
- Listen to any port number on the receiving side
- Dial the destination port number on the sender

Example of use:

- Listen on TCP 1987 port on the receiving side

$ rcp listen -l :1987 -o save_filename

- Send file from sender

$ rcp send -d 10.10.10.10:1987 -i input_filename`,

	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", cfgFile, "config file (default is $HOME/.rcp.yaml)")
	rootCmd.PersistentFlags().IntVar(&r.MaxBufNum, "maxBufNum", r.MaxBufNum, "Maximum number of buffers (with thread copy mode)")
	rootCmd.PersistentFlags().IntVar(&r.BufSize, "bufSize", r.BufSize, "Buffer size(with thread copy mode)")
	rootCmd.PersistentFlags().BoolVarP(&r.SingleThread, "singlThread", "s", r.SingleThread, "Single thread mode")
	rootCmd.PersistentFlags().StringVar(&dummyInputString, "dummyInput", dummyInputString, "dummy input mode data size (ex: 100MB, 4K, 10g)")
	rootCmd.PersistentFlags().BoolVar(&r.DummyOutput, "dummyOutput", r.DummyOutput, "dummy output mode")
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".rcp" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".rcp")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
