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
	"github.com/dmorgan81/buzzel/pkg/cache"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var memCmd = &cobra.Command{
	Use:           "mem",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info().Msg("mem cache")
		return runServer(cache.NewMemCache())
	},
}

func init() {
	rootCmd.AddCommand(memCmd)
}
