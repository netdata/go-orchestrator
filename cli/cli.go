package cli

import (
	"strconv"

	"github.com/jessevdk/go-flags"
)

// Option defines command line options.
type Option struct {
	UpdateEvery int
	Debug       bool     `short:"d" long:"debug" description:"debug mode"`
	Module      string   `short:"m" long:"modules" description:"modules name" default:"all"`
	ConfigDir   []string `short:"c" long:"config" description:"config dir"`
	Version     bool     `short:"v" long:"version" description:"display the version and exit"`
}

// Parse returns parsed command-line flags in Option struct
func Parse(args []string) (*Option, error) {
	opt := &Option{
		UpdateEvery: 1,
	}
	parser := flags.NewParser(opt, flags.Default)
	parser.Name = "orchestrator"
	parser.Usage = "[OPTIONS] [update every]"

	rest, err := parser.ParseArgs(args)
	if err != nil {
		return nil, err
	}

	if len(rest) > 1 {
		if opt.UpdateEvery, err = strconv.Atoi(rest[1]); err != nil {
			return nil, err
		}
	}

	return opt, nil
}
