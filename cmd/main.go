package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/alecthomas/kong"
	"github.com/spf13/viper"

	"nitro/markdown-link-check/internal"
)

type config struct {
	Ignore []string `mapstructure:"ignore"`
}

func main() {
	var params struct {
		Path   string `help:"Path to be processed" required:"true" arg:"true" type:"string"`
		Config string `help:"Path to the configuration file." required:"true" short:"c" type:"string"`
	}
	kong.Parse(&params, kong.Name("markdown-link-check"))

	client, err := configClient(params.Config)
	if err != nil {
		handleError("fail to configure the client: %s", err.Error())
	}
	client.Path = params.Path

	hasInvalidLinks, err := client.Run(executionContext())
	if err != nil {
		handleError("fail at client execution: %s", err.Error())
	}
	if hasInvalidLinks {
		os.Exit(1)
	}
}

func configClient(path string) (internal.Client, error) {
	f, err := os.Open(path)
	if err != nil {
		return internal.Client{}, fmt.Errorf("fail to open the config file: %w", err)
	}
	defer f.Close()

	var viper = viper.New()
	viper.SetConfigType("yaml")
	if err := viper.ReadConfig(f); err != nil {
		return internal.Client{}, fmt.Errorf("fail to read the configuration file: %w", err)
	}

	var cfg config
	if err := viper.Unmarshal(&cfg); err != nil {
		return internal.Client{}, fmt.Errorf("fail to unmarshal the configuration: %w", err)
	}

	return internal.Client{
		Ignore: cfg.Ignore,
	}, nil
}

func handleError(mask string, params ...interface{}) {
	fmt.Printf(mask+"\n", params...)
	os.Exit(1)
}

func executionContext() context.Context {
	ctx, ctxCancel := context.WithCancel(context.Background())
	go func() {
		chSignal := make(chan os.Signal, 1)
		signal.Notify(chSignal, os.Interrupt)
		<-chSignal
		ctxCancel()
	}()
	return ctx
}
