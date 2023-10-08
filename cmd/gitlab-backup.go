// gitlab-backup
// Copyright (C) 2021  Sylvain Gaunet

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sgaunet/gitlab-backup/pkg/app"
	"github.com/sgaunet/gitlab-backup/pkg/config"
)

var version string = "development"

func printVersion() {
	fmt.Println(version)
}

func main() {
	var cfgFile string
	var vOption bool
	var helpOption bool
	var printCfg bool
	var err error

	// Parameters treatment
	flag.StringVar(&cfgFile, "c", "", "configuration file")
	flag.BoolVar(&printCfg, "cfg", false, "print configuration")
	flag.BoolVar(&vOption, "v", false, "Get version")
	flag.BoolVar(&helpOption, "h", false, "help")
	flag.Parse()

	if helpOption {
		flag.Usage()
		os.Exit(0)
	}

	if vOption {
		printVersion()
		os.Exit(0)
	}

	if printCfg {
		c, err := config.NewConfigFromEnv()
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		c.Usage()

		fmt.Println("--------------------------------------------------")
		fmt.Println("Gitlab-backup configuration:")
		fmt.Printf("%+v\n", c)
		os.Exit(0)
	}

	app, err := app.NewApp(cfgFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	l := initTrace(os.Getenv("DEBUGLEVEL"))
	app.SetLogger(l)
	err = app.Run()

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
