package sensors

import (
	"flag"

	"github.com/pointer2null/weather/env"
	logger "github.com/sirupsen/logrus"
	"periph.io/x/periph/conn/i2c"
	"periph.io/x/periph/conn/i2c/i2creg"
	"periph.io/x/periph/host"
)

type Sensors struct {
	Atm    *atmosphere
	Rain   *rainmeter
	Wind   *Anemometer
	IMU    *IMU
	Closer *i2c.BusCloser
}

func InitSensors(args *env.Args) *Sensors {
	s := &Sensors{}

	if _, err := host.Init(); err != nil {
		logger.Fatalf("Failed to init i2c bus [%v]", err)
		return nil
	}
	i2cbus := flag.String("bus", "1", "I²C bus (/dev/i2c-1)")
	logger.Infof("Opening I2C bus [%v]", i2cbus)
	closer, err := i2creg.Open(*i2cbus)
	if err != nil {
		logger.Fatalf("failed to open I²C: %v", err)
		_ = closer.Close()
		return nil
	}
	s.Closer = &closer
	bus := i2c.Bus(closer)

	if *args.AtmosphericEnabled {
		s.Atm = NewAtmosphere(&bus, *args)
	}
	if *args.RainEnabled {
		s.Rain = NewRainmeter(&bus, *args)
	}
	if *args.WindEnabled {
		s.Wind = NewAnemometer(&bus, *args)
	}
	if *args.Imuon {
		s.IMU = NewIMU(&bus, *args)
	}
	return s
}
