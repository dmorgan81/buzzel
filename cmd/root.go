/*
Copyright © 2021 David Morgan <dmorgan81@gmail.com>

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
package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var rootCmd = &cobra.Command{
	Use: "buzzel",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		level, err := zerolog.ParseLevel(viper.GetString("log.level"))
		if err != nil {
			return err
		}
		zerolog.SetGlobalLevel(level)

		if viper.GetBool("log.pretty") {
			log.Logger = log.Output(zerolog.NewConsoleWriter())
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	flags := rootCmd.Flags()
	flags.String("log.level", "info", "")
	flags.Bool("log.pretty", true, "")

	viper.BindPFlags(flags)
}

func initConfig() {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("buzzel")
}
