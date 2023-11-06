/*
Copyright © 2023 Tom Manterfield <tom@hiphops.io>

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
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	addkeyShortDesc = "Add your hiphops keyfile"
	addkeyLongDesc  = `Add your hiphops keyfile to the default location (or --keyfile path if set).

	The hiphops key can be found on your hiphops.io account page.
	If you are not the account owner, they must provide it to you.
	
	Warning: If you have a keyfile already present in the default location, it will be overwritten.

	Note: Adding your keyfile can be done manually by saving the key contents to file
	either in the default location ($ROOTDIR/hiphops.key) or a location of your choosing and
	passing in the --keyfile=MYPATH flag. The outcome will be identical.
	`
)

// addkeyCmd is a helper function to load a hiphops key into the correct location
func addkeyCmd() *cobra.Command {
	addkeyCmd := &cobra.Command{
		Use:   "addkey",
		Short: addkeyShortDesc,
		Long:  addkeyLongDesc,
		RunE:  addkey,
	}

	addkeyCmd.Flags().String("keydata", "", "The hiphops key as taken from the account page")
	addkeyCmd.MarkFlagRequired("keydata")
	viper.BindPFlag("keydata", addkeyCmd.Flags().Lookup("keydata"))

	return addkeyCmd
}

func addkey(cmd *cobra.Command, args []string) error {
	logger := cmdLogger()

	err := overwriteFile(viper.GetString("keyfile"), []byte(viper.GetString("keydata")))
	if err != nil {
		logger.Error().Err(err).Msg("Failed to write keyfile")
		return err
	}

	return nil
}

func overwriteFile(filepath string, content []byte) error {
	writeFile, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer writeFile.Close()

	writeFile.Truncate(0)
	writeFile.Seek(0, 0)
	writeFile.Write(content)
	writeFile.Sync()

	return nil
}
