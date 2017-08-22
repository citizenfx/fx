package main

import (
	"os"

	"gopkg.in/urfave/cli.v1"
)

func main() {
	app := cli.NewApp()
	app.Name = "fx"
	app.Usage = "the CitizenFX manager"
	app.Version = "1.0.0"
	app.Commands = []cli.Command{
		{
			Name:   "get",
			Usage:  "Gets a resource and installs it into the resource tree",
			Action: cmdGet,
			Flags: []cli.Flag{
				cli.BoolFlag{Name: "force,f", Usage: "Force update, even if unclean"},
			},
		},
		{
			Name:   "add",
			Usage:  "Adds a resource to the server",
			Action: cmdAdd,
			Flags: []cli.Flag{
				cli.BoolFlag{Name: "force,f", Usage: "Force update, even if unclean"},
				cli.StringFlag{Name: "config-file,c", Value: "resources.cfg", Usage: "The resource configuration file to add to"},
			},
		},
		{
			Name:   "new",
			Usage:  "Makes a new server instance in the current directory",
			Action: cmdNew,
			Flags: []cli.Flag{
				cli.IntFlag{Name: "port,p", Value: 30120, Usage: "The port for the new server"},
				cli.BoolFlag{Name: "disable-default", Usage: "Disable loading the default 'fivem' resource"},
			},
		},
	}

	app.Run(os.Args)
}
