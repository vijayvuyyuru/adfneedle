package main

import (
	"adfneedle/models"
	"context"

	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/module"
	"go.viam.com/utils"
)

func main() {
	utils.ContextualMain(mainWithArgs, module.NewLoggerFromArgs("adfneedle"))
}

func mainWithArgs(ctx context.Context, args []string, logger logging.Logger) error {
	adfneedle, err := module.NewModuleFromArgs(ctx)
	if err != nil {
		return err
	}

	if err = adfneedle.AddModelFromRegistry(ctx, sensor.API, models.Sensor); err != nil {
		return err
	}

	err = adfneedle.Start(ctx)
	defer adfneedle.Close(ctx)
	if err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}
