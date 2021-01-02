package main

import (
	"time"

	logger "github.com/sirupsen/logrus"
	"periph.io/x/periph/conn/physic"
)

/*
Measuring gusts and wind intensity

Because wind is an element that varies rapidly over very short periods of
time it is sampled at high frequency (every 0.25 sec) to capture the intensity
of gusts, or short-lived peaks in speed, which inflict greatest damage in
storms. The gust speed and direction are defined by the maximum three second
average wind speed occurring in any period.

A better measure of the overall wind intensity is defined by the average speed
and direction over the ten minute period leading up to the reporting time.
Mean wind over other averaging periods may also be calculated. A gale is
defined as a surface wind of mean speed of 34-40 knots, averaged over a period
of ten minutes. Terms such as 'severe gale', 'storm', etc are also used to
describe winds of 41 knots or greater.

How do we measure the wind.

The anemometer I use generates 1 pulse per revolution and the specifications states
that equates to 1.429 MPH. This will need to be confirmed and calibrated at some time.

When the readWindData function is called from main as a go routine, it starts two other
threads: monitorWindGPIO and processWindSpeed.

monitorWindGPIO sits in a forever loop and waits for the GPIO pin to be triggered. On each
tick it calculates the instantanious wind speed based on the time since the last tick was
recorded.

The second thread processWindSpeed has a ticker that fires every 250ms. On each tick it records
instantanious wind speed. It then clear the value - if it didn't we would not know if the wind
stopped! If we have collected 4 values then we calculate the max and average of those values.
The max gives us the value for the wind gust and the avg for that second is recorded in another
array. When we have accumulated 60 values we work out the average wind speed for the last minute.

Values for windspeed, gust and direction are stored in local variable for the local web server
and in the prometeus guages for further processing.
*/

const (
	// 1 tick/second = 1.492MPH wind
	mphPerTick float64 = 1.429
)

var (
	livespeed float64 = 0
)

func (s *weatherstation) readWindData() {
	go s.monitorWindGPIO()
	go s.processWindSpeed()
	for range time.Tick(time.Second * 10) {
		s.readWindDirection()
	}
}

var pcount = 0
var wsum   = 0.0
var wmax   = 0.0

// monitorWindGPIO watches the gpio port on tick calculate the instantanious wind speed.
// WaitForEdge returns immediately IF another pulse has arrived since the last call. 
// need to make sure any queue is cleared before we restart the loop
func (w *weatherstation) monitorWindGPIO() {
	logger.Info("Starting wind sensor")
	defer func() { _ = (*w.s.windpin).Halt() }()
	lasttick := time.Now().Add(time.Duration(-1) * time.Second)
	var edge time.Time
	for {
		(*w.s.windpin).WaitForEdge(-1)
		edge = time.Now()
		period := edge.Sub(lasttick).Seconds()
		if period != 0 {
			freq := float64(1 / period)
			speed := freq * mphPerTick
			livespeed = speed
			//logger.Infof(">>>>> Wind pulse p[%.4f] f[%.4f] s[%2.2f]", period, freq, speed)
			lasttick = edge
			pcount++
			wsum += livespeed
			if livespeed > wmax {
				wmax = livespeed
			}
		}
		// need to clear out any pulses that have arrived since
		// almost certain to be junk unless we're in a hurracane on jupiter
		var crud bool
		for {
			// wait for edge with small timeout
			// "Returns true if an edge was detected during or before this call"
			crud = (*w.s.windpin).WaitForEdge(time.Microsecond)
			// if we hit timeout crud = false, then we exit
			if !crud {
				break
			}
		}
	}
}

func (w *weatherstation) processWindSpeed() {
	avg := 0.0
	// initial values
	windspeed.Set(livespeed)
	windgust.Set(livespeed)
	// start ticker
	for range time.Tick(time.Minute) {		
		if pcount > 0 {
			avg = wsum / float64(pcount)
		}
		w.windSpeedAvg = avg
		windspeed.Set(avg)
		if avg > wmax { // TODO !!FUDGE!! occasionally see avg > max, since we're not thread safe...
			w.windGust = avg
			windgust.Set(avg)
		} else {
			w.windGust = wmax
			windgust.Set(wmax)
		}
		logger.Infof("Wind Avg [%.2f] Gust [%.2f]", avg, wmax)
		wmax = 0.0
		avg = 0.0
	}
}

func (w *weatherstation) readWindDirection() {
	sample, err := (*w.s.windDir).Read()
	if err != nil {
		logger.Errorf("Error reading wind direction value [%v]", err)
		sample.Raw = 0
	}
	w.windVolts = float64(sample.V) / float64(physic.Volt)
	w.windDirection = voltToDegrees(w.windVolts)
	logger.Debugf("Volt [%v], Dir [%v]", w.windVolts, w.windDirection)

	// prometheus data
	logger.Debugf("Setting winddir [%v]", w.s.windDir)
	windDirection.Set(w.windDirection)
}

func voltToDegrees(v float64) float64 {
	// this is based on the sensor datasheet that gives a list of voltages for each direction when set up according
	// to the circuit given. Have noticed the output isn't that accurate relative to the sensor direction...
	switch {
	case v < 0.365:
		return 112.5
	case v < 0.430:
		return 67.5
	case v < 0.535:
		return 90.0
	case v < 0.760:
		return 157.5
	case v < 1.045:
		return 135.0
	case v < 1.295:
		return 202.5
	case v < 1.690:
		return 180.0
	case v < 2.115:
		return 22.5
	case v < 2.590:
		return 45.0
	case v < 3.005:
		return 247.5
	case v < 3.225:
		return 225.0
	case v < 3.635:
		return 337.5
	case v < 3.940:
		return 0
	case v < 4.185:
		return 292.5
	case v < 4.475:
		return 315.0
	default:
		return 270.0
	}
}
