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

	"github.com/masahide/rcp/pkg/bytesize"
	"github.com/spf13/cobra"
)

// sendCmd represents the send command
var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send files",
	Long: `Send a file to a listening TCP port
example:

$ rcp send -d 10.10.10.10:1987 -i input_filename`,
	Run: func(cmd *cobra.Command, args []string) {
		r.DummyInput = int64(bytesize.MustParse(dummyInputString))
		if len(r.DialAddr) == 0 && r.DummyInput == 0 {
			log.Fatal("--dialAddr(-d) flag or --dummyInput flag required")
		}
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
	rootCmd.AddCommand(sendCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// sendCmd.PersistentFlags().String("foo", "", "A help for foo")
	sendCmd.PersistentFlags().StringVarP(&r.Input, "input", "i", r.Input, "input filename")
	sendCmd.PersistentFlags().StringVarP(&r.DialAddr, "dialAddr", "d", r.DialAddr, "dial address (ex: 198.51.100.1:1987 )")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// sendCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
