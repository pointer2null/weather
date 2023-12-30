package main

import (
	"time"

	"github.com/pointer2null/weather/buffer"
	"github.com/pointer2null/weather/constants"
	logger "github.com/sirupsen/logrus"
)

const (
	hourRateMins int = 10 // number of minutes to average for hourly rate
	RainBuffer       = "rain"
)

func (w *weatherstation) StartRainMonitor() {
	w.setupRainBuffers()
	go w.readRainData()
}

// once per minute the number of bucket tips are read and we store this in the minute buffer
// once per hour on the overflow we calculate the min/max for that hour and save to their buffers
func (w *weatherstation) readRainData() {
	for range time.Tick(time.Minute) {
		count := 0 //w.s.GetRainCount()

		// add this to the rain minute buffer
		rbuff := w.data.GetBuffer(RainBuffer)
		rbuff.AddItem(float64(count))

		// Does this belong here? Or should this file just be about recording the data?
		mmLastMinute := float64(count) * constants.MMPerBucketTip
		tips, _, _ := rbuff.SumMinMaxLast(hourRateMins)
		tenMinSum_mm := tips * buffer.Sum(constants.MMPerBucketTip)
		hourRate_mm := float64(tenMinSum_mm) * 60 / float64(hourRateMins)

		Prom_rainRatePerHour.Set(hourRate_mm)

		// day totals - get the hour sum
		_, _, _, s := rbuff.GetAutoSum().GetAverageMinMaxSum()
		day := float64(s) * constants.MMPerBucketTip
		Prom_rainDayTotal.Set(day)

		logger.Infof("Rain [%.2f] -> hourly rate [%.2f], 24 hour total [%.2f]", mmLastMinute, hourRate_mm, s)
	}
}

func (w *weatherstation) setupRainBuffers() {

	rainMinuteBuffer := buffer.NewBuffer(60)

	// add on auto hour buffers to track day values
	// rainMinimumHourBuffer := buffer.NewBuffer(24)
	// rainMinuteBuffer.SetAutoMinimum(rainMinimumHourBuffer)
	// rainMaximumHourBuffer := buffer.NewBuffer(24)
	// rainMinuteBuffer.SetAutoMaximum(rainMaximumHourBuffer)
	rainSumHourBuffer := buffer.NewBuffer(24)
	rainMinuteBuffer.SetAutoSum(rainSumHourBuffer)

	w.data.AddBuffer(RainBuffer, rainMinuteBuffer)
}
