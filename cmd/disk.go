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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dmorgan81/buzzel/pkg/cache/disk"
	"github.com/rs/zerolog/log"
)

var diskCmd = &cobra.Command{
	Use:           "disk",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := viper.GetString("cache.disk.dir")
		log.Info().Str("cache dir", dir).Send()
		return runServer(disk.Cache(dir))
	},
}

func init() {
	rootCmd.AddCommand(diskCmd)

	flags := diskCmd.Flags()
	flags.String("cache.disk.dir", "./", "")

	viper.BindPFlags(flags)
}
