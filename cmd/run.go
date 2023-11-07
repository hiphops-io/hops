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
	"github.com/spf13/cobra"
)

const (
	runShortDesc = "Run your custom commands"
	runLongDesc  = `Run a command defined within your hops config.

Trigger complex, multi-step automation workflows using the full power of hops.`
)

// runCmd runs custom/user defined commands
func runCmd() *cobra.Command {
	runCmd := &cobra.Command{
		Use:                "run COMMAND_NAME [flags]",
		Short:              runShortDesc,
		Long:               runLongDesc,
		Args:               cobra.ExactArgs(1),
		ValidArgs:          []string{"command"},
		DisableFlagParsing: true,
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	return runCmd
}
