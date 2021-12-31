package service

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Jeffail/benthos/v3/internal/bundle"
	"github.com/Jeffail/benthos/v3/internal/docs"
	"github.com/Jeffail/benthos/v3/lib/cache"
	"github.com/Jeffail/benthos/v3/lib/config"
	"github.com/Jeffail/benthos/v3/lib/input"
	"github.com/Jeffail/benthos/v3/lib/output"
	"github.com/Jeffail/benthos/v3/lib/pipeline"
	"github.com/Jeffail/benthos/v3/lib/processor"
	"github.com/Jeffail/benthos/v3/lib/ratelimit"
	uconfig "github.com/Jeffail/benthos/v3/lib/util/config"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

func addExpression(conf *config.Type, expression string) error {
	var inputTypes, processorTypes, outputTypes []string
	componentTypes := strings.Split(expression, "/")
	for i, str := range componentTypes {
		for _, t := range strings.Split(str, ",") {
			if t = strings.TrimSpace(t); len(t) > 0 {
				switch i {
				case 0:
					inputTypes = append(inputTypes, t)
				case 1:
					processorTypes = append(processorTypes, t)
				case 2:
					outputTypes = append(outputTypes, t)
				default:
					return errors.New("more component separators than expected")
				}
			}
		}
	}

	if lInputs := len(inputTypes); lInputs == 1 {
		t := inputTypes[0]
		if _, exists := bundle.AllInputs.DocsFor(t); exists {
			conf.Input.Type = t
		} else {
			return fmt.Errorf("unrecognised input type '%v'", t)
		}
	} else if lInputs > 1 {
		conf.Input.Type = input.TypeBroker
		for _, t := range inputTypes {
			c := input.NewConfig()
			if _, exists := bundle.AllInputs.DocsFor(t); exists {
				c.Type = t
			} else {
				return fmt.Errorf("unrecognised input type '%v'", t)
			}
			conf.Input.Broker.Inputs = append(conf.Input.Broker.Inputs, c)
		}
	}

	for _, t := range processorTypes {
		c := processor.NewConfig()
		if _, exists := bundle.AllProcessors.DocsFor(t); exists {
			c.Type = t
		} else {
			return fmt.Errorf("unrecognised processor type '%v'", t)
		}
		conf.Pipeline.Processors = append(conf.Pipeline.Processors, c)
	}

	if lOutputs := len(outputTypes); lOutputs == 1 {
		t := outputTypes[0]
		if _, exists := bundle.AllOutputs.DocsFor(t); exists {
			conf.Output.Type = t
		} else {
			return fmt.Errorf("unrecognised output type '%v'", t)
		}
	} else if lOutputs > 1 {
		conf.Output.Type = output.TypeBroker
		for _, t := range outputTypes {
			c := output.NewConfig()
			if _, exists := bundle.AllOutputs.DocsFor(t); exists {
				c.Type = t
			} else {
				return fmt.Errorf("unrecognised output type '%v'", t)
			}
			conf.Output.Broker.Outputs = append(conf.Output.Broker.Outputs, c)
		}
	}
	return nil
}

type minimalCreateConfig struct {
	Input              input.Config       `json:"input" yaml:"input"`
	Pipeline           pipeline.Config    `json:"pipeline" yaml:"pipeline"`
	Output             output.Config      `json:"output" yaml:"output"`
	ResourceCaches     []cache.Config     `json:"cache_resources,omitempty" yaml:"cache_resources,omitempty"`
	ResourceRateLimits []ratelimit.Config `json:"rate_limit_resources,omitempty" yaml:"rate_limit_resources,omitempty"`
}

func createCliCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a new Benthos config",
		Description: `
   Prints a new Benthos config to stdout containing specified components
   according to an expression. The expression must take the form of three
   comma-separated lists of inputs, processors and outputs, divided by
   forward slashes:

   benthos create stdin/bloblang,awk/nats
   benthos create file,http_server/protobuf/http_client

   If the expression is omitted a default config is created.`[4:],
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "small",
				Aliases: []string{"s"},
				Value:   false,
				Usage:   "Print only the main components of a Benthos config (input, pipeline, output) and omit all fields marked as advanced.",
			},
		},
		Action: func(c *cli.Context) error {
			if expression := c.Args().First(); len(expression) > 0 {
				if err := addExpression(&conf, expression); err != nil {
					fmt.Fprintf(os.Stderr, "Generate error: %v\n", err)
					os.Exit(1)
				}
			}

			var filter docs.FieldFilter
			var iconf interface{} = conf

			if c.Bool("small") {
				iconf = minimalCreateConfig{
					Input:              conf.Input,
					Pipeline:           conf.Pipeline,
					Output:             conf.Output,
					ResourceCaches:     conf.ResourceCaches,
					ResourceRateLimits: conf.ResourceRateLimits,
				}

				filter = func(spec docs.FieldSpec) bool {
					return !spec.IsAdvanced
				}
			}

			var node yaml.Node
			err := node.Encode(iconf)
			if err == nil {
				err = config.Spec().SanitiseYAML(&node, docs.SanitiseConfig{
					RemoveTypeField:  true,
					RemoveDeprecated: true,
					ForExample:       true,
					Filter:           filter,
				})
			}
			if err == nil {
				var configYAML []byte
				if configYAML, err = uconfig.MarshalYAML(node); err == nil {
					fmt.Println(string(configYAML))
				}
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Generate error: %v\n", err)
				os.Exit(1)
			}
			return nil
		},
	}
}
