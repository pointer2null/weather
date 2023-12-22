package main

import (
	"time"

	"github.com/pointer2null/weather/utils"
	logger "github.com/sirupsen/logrus"
)

const (
	TempBuffer     = "temperature"
	HumidityBuffer = "humidity"
	PressureBuffer = "pressurehPa"
)

func (w *weatherstation) StartAtmosphericMonitor() {
	logger.Info("Starting atmosphere monitors")
	// set the the required buffers
	w.setupTemperatureBuffers()
	w.setupHumidityBuffers()
	w.SetupPressurehPaBuffers()

	duration := time.Minute
	if w.testMode {
		logger.Info("Atmosphereic poll set to 5 seconds")
		duration = time.Second * 5
	}

	// sample and store sensor data
	for range time.Tick(duration) {
		if w.testMode {
			logger.Info("Reading atmospheric data ...")
		}
		t := w.s.GetTemperature()
		hPa, rh := w.s.GetHumidityAndPressure()
		logger.Infof("Temperature [%3.2f], Pressure [%3.2f], Humidity [%3.2f]", t, hPa, rh)

		w.data.GetBuffer(TempBuffer).AddItem(float64(t))
		Prom_temperature.Set(float64(t))

		w.data.GetBuffer(HumidityBuffer).AddItem(float64(rh))
		w.data.GetBuffer(PressureBuffer).AddItem(float64(hPa))
		Prom_atmPresure.Set(float64(hPa))
		Prom_humidity.Set(float64(rh))
	}
}

func (w *weatherstation) setupTemperatureBuffers() {

	temperatureMinuteBuffer := utils.NewBuffer(60)

	// tempAvgHourBuffer := utils.NewBuffer(24)
	// temperatureMinuteBuffer.SetAutoAverage(tempAvgHourBuffer)

	// tempMinHourBuffer := utils.NewBuffer(24)
	// temperatureMinuteBuffer.SetAutoMinimum(tempMinHourBuffer)

	// tempMaxHourBuffer := utils.NewBuffer(24)
	// temperatureMinuteBuffer.SetAutoMaximum(tempMaxHourBuffer)

	w.data.AddBuffer(TempBuffer, temperatureMinuteBuffer)
}

func (w *weatherstation) setupHumidityBuffers() {

	humidityMinuteBuffer := utils.NewBuffer(60)

	// humidityAvgHourBuffer := utils.NewBuffer(24)
	// humidityMinuteBuffer.SetAutoAverage(humidityAvgHourBuffer)

	// humidityMinHourBuffer := utils.NewBuffer(24)
	// humidityMinuteBuffer.SetAutoMinimum(humidityMinHourBuffer)

	// humidityMaxHourBuffer := utils.NewBuffer(24)
	// humidityMinuteBuffer.SetAutoMaximum(humidityMaxHourBuffer)

	w.data.AddBuffer(HumidityBuffer, humidityMinuteBuffer)
}

func (w *weatherstation) SetupPressurehPaBuffers() {

	pressurehPaMinuteBuffer := utils.NewBuffer(60)

	// pressurehPaAvgHourBuffer := utils.NewBuffer(24)
	// pressurehPaMinuteBuffer.SetAutoAverage(pressurehPaAvgHourBuffer)

	// pressurehPaMinHourBuffer := utils.NewBuffer(24)
	// pressurehPaMinuteBuffer.SetAutoMinimum(pressurehPaMinHourBuffer)

	// pressurehPaMaxHourBuffer := utils.NewBuffer(24)
	// pressurehPaMinuteBuffer.SetAutoMaximum(pressurehPaMaxHourBuffer)

	w.data.AddBuffer(PressureBuffer, pressurehPaMinuteBuffer)
}
