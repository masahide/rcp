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
	"log"

	"github.com/spf13/cobra"
)

// listenCmd represents the listen command
var listenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Listen to the file receiving port command",
	Long: `Listen to the file receiving port command
example:

$ rcp listen -a 0.0.0.0:1987 -o outputfile
`,
	Run: func(cmd *cobra.Command, args []string) {
		_, err := r.ReadWrite()
		if err != nil {
			log.Println(err)
		}
		fmt.Println(r.SpeedDashboard.Input.Title)
		fmt.Println(r.SpeedDashboard.Output.Title)
		fmt.Println(r.SpeedDashboard.Buffer.Title)
		fmt.Println(r.SpeedDashboard.Progress.Title)
	},
}

func init() {
	rootCmd.AddCommand(listenCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// listenCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// listenCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	listenCmd.PersistentFlags().StringVarP(&r.ListenAddr, "listenAddr", "l", r.ListenAddr, "listen address")
	listenCmd.PersistentFlags().StringVarP(&r.Output, "output", "o", r.Output, "output filename")
	listenCmd.PersistentFlags().Int64Var(&r.DummyInput, "dummyInput", r.DummyInput, "dummy input mode data size")
	listenCmd.PersistentFlags().BoolVar(&r.DummyOutput, "dummyOutput", r.DummyOutput, "dummy output mode")
	//flag.BoolVar(&discard, "discard", discard, "discard output")
	//flag.StringVar(&input, "i", input, "input filename")

}
