package main

import (
	"context"
	"log"
	"time"

	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/components/servo"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/robot/client"
)

const servoMaxAngle = 115.0

var logger = logging.NewLogger("client")

func main() {

	for {
		setPosition()
		time.Sleep(30 * time.Minute)
	}
}

func setPosition() {
	logger.Info("in set poistion")
	machine, err := client.New()
	if err != nil {
		logger.Fatal(err)
	}

	defer machine.Close(context.Background())
	// servo-1
	servo1, err := servo.FromRobot(machine, "servo-2")
	if err != nil {
		logger.Fatal(err)
		return
	}

	// sensor-1
	sensor1, err := sensor.FromRobot(machine, "sensor-1")
	if err != nil {
		logger.Fatal(err)
		return
	}
	sensor1ReturnValue, err := sensor1.Readings(context.Background(), map[string]any{})
	if err != nil {
		logger.Fatal(err)
		return
	}
	usageRaw, ok := sensor1ReturnValue["usage"]
	if !ok {
		logger.Fatal("cannot get usage from sensor", "return", sensor1ReturnValue)
		return
	}
	usage := usageRaw.(float64)

	servoPosition := servoMaxAngle - usage*servoMaxAngle
	err = servo1.Move(context.Background(), uint32(servoPosition), map[string]any{})
	if err != nil {
		log.Fatal(err)
		return
	}
}
