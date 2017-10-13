package main

import (
	"fmt"
	"github.com/urfave/cli"
	"math"
	"os"
)

var VERSION = "0.0.0-dev"
var REVISION = "HEAD"
var BUILD_TIME = ""

func main() {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("version=%s revision=%s built on=%s\n", VERSION, REVISION, BUILD_TIME)
	}

	app := cli.NewApp()
	app.Name = "gitlab-ci-linter"
	app.Version = fmt.Sprintf("%s (%s)", VERSION, REVISION[:int(math.Min(float64(len(REVISION)), 7))])
	app.Authors = []cli.Author{
		{Name: "Olivier ROBARDET"},
	}
	app.Usage = "lint your .gitlab-ci.yml using the Gitlab lint API"
	app.EnableBashCompletion = true
	app.Commands = []cli.Command{
		{
			Name:    "version",
			Aliases: []string{"v"},
			Usage:   "Show version information",
			Action: func(c *cli.Context) {
				cli.ShowVersion(c)
			},
		},
	}
	app.Action = func(c *cli.Context) error {
		fmt.Println("Lint the ci")
		return nil
	}

	app.Run(os.Args)
}
