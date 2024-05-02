package main

// import (
// 	"github.com/urfave/cli/v2"
// 	"github.com/urfave/cli/v2/altsrc"

// 	"github.com/hiphops-io/hops/dsl"
// )

// const validateDescription = `Validate automations in a given directory, returning
// a JSON object describing diagnostic errors from parsing automations, if any occurred.

// Basic usage (uses default automations dir ~/.hops/):
// 	hops validate

// Validate specific directory:
// 	hops validate -H /my/dir/containing/automation/dirs
// `

// func initValidateCommand(commonFlags []cli.Flag) *cli.Command {
// 	validateFlags := initValidateFlags(commonFlags)
// 	before := optionalYamlSrc(validateFlags)

// 	return &cli.Command{
// 		Name:        "validate",
// 		Usage:       "Validate automations",
// 		Description: validateDescription,
// 		Before:      before,
// 		Flags:       validateFlags,
// 		Action: func(c *cli.Context) error {
// 			return dsl.ValidateDir(c.String("hops"), c.Bool("pretty"))
// 		},
// 	}
// }

// func initValidateFlags(commonFlags []cli.Flag) []cli.Flag {
// 	validateFlags := []cli.Flag{
// 		altsrc.NewBoolFlag(
// 			&cli.BoolFlag{
// 				Name:    "pretty",
// 				Aliases: []string{"validate.pretty"},
// 				Usage:   "Whether to pretty print (indent and colorize) the JSON validation output",
// 			},
// 		),
// 	}

// 	return append(validateFlags, commonFlags...)
// }
