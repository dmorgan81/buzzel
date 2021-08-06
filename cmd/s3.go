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
	"errors"

	"github.com/dmorgan81/buzzel/pkg/cache/s3"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var s3Cmd = &cobra.Command{
	Use:           "s3",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		bucket := viper.GetString("cache.s3.bucket")
		if bucket == "" {
			return errors.New("s3 bucket is required")
		}
		log.Info().Str("cache bucket", bucket).Send()

		cache, err := s3.NewS3Cache(bucket)
		if err != nil {
			return err
		}
		return runServer(cache)
	},
}

func init() {
	rootCmd.AddCommand(s3Cmd)

	flags := s3Cmd.Flags()
	flags.String("cache.s3.bucket", "", "")

	viper.BindPFlags(flags)
}
