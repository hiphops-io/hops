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
	"path/filepath"

	"github.com/hiphops-io/hops/logs"
	"github.com/urfave/cli/v2"
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

func initAddKeyCommand(commonFlags []cli.Flag) *cli.Command {
	addkeyFlags := []cli.Flag{
		&cli.StringFlag{
			Name:     "keydata",
			Usage:    "The hiphops key as taken from the account page",
			Required: true,
		},
	}
	addkeyFlags = append(addkeyFlags, commonFlags...)
	before := optionalYamlSrc(addkeyFlags)

	return &cli.Command{
		Name:        "addkey",
		Usage:       addkeyShortDesc,
		Description: addkeyLongDesc,
		Before:      before,
		Flags:       addkeyFlags,
		Action: func(c *cli.Context) error {
			logger := logs.InitLogger(c.Bool("debug"))

			err := overwriteFile(c.String("keyfile"), []byte(c.String("keydata")))
			if err != nil {
				logger.Error().Err(err).Msg("Failed to write keyfile")
				return err
			}

			return nil
		},
	}
}

func overwriteFile(fileNamePath string, content []byte) error {
	// Create all directories in the path if they do not exist
	dirPath := filepath.Dir(fileNamePath)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}

	// Open the file with flags to create it if it doesn't exist, and for read-write
	writeFile, err := os.OpenFile(fileNamePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer writeFile.Close()

	// Write the content to the file
	if _, err := writeFile.Write(content); err != nil {
		return err
	}

	return writeFile.Sync()
}
