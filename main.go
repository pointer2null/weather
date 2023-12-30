package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"database/sql"

	_ "github.com/lib/pq"
	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"

	"github.com/pointer2null/weather/constants"
	"github.com/pointer2null/weather/data"
	"github.com/pointer2null/weather/db/postgres"
	"github.com/pointer2null/weather/led"
	"github.com/pointer2null/weather/sensors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	logger "github.com/sirupsen/logrus"
)

const version = "GRB-Weather-2.0.0"

const (
	host     = "192.168.1.212"
	port     = 5432
	user     = "weather"
	password = "weather01."
	dbname   = "weather"
)

type weatherstation struct {
	s            *sensors.Sensors
	data         *data.WeatherData
	dbq          *postgres.Queries
	testMode     bool
	HeartbeatLed *led.LED
}

type webdata struct {
	TimeNow      string  `json:"time"`
	TempHiRes    float64 `json:"hiResTemp_C"`
	Humidity     float64 `json:"humidity_RH"`
	Pressure     float64 `json:"pressure_hPa"`
	RainHr       float64 `json:"rain_mm_hr"`
	RainRate     float64 `json:"rain_rate"`
	WindDir      float64 `json:"wind_dir"`
	WindSpeed    float64 `json:"wind_speed"`
	WindSpeedAvg float64 `json:"wind_speed_avg"`
	WindGust     float64 `json:"wind_gust"`
}

var Prom_atmPresure = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "atmospheric_pressure",
		Help: "Atmospheric pressure hPa",
	},
)

var Prom_rainRatePerHour = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "rain_hour_rate",
		Help: "The rain rate based on the last 5 minuntes",
	},
)

var Prom_rainDayTotal = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "rain_day",
		Help: "The rain total today (9.01am - 9am)",
	},
)

var Prom_humidity = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "relative_humidity",
		Help: "Relative Humidity",
	},
)

var Prom_temperature = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "temperature",
		Help: "Temperature C",
	},
)

var Prom_windspeed = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "windspeed",
		Help: "Average Wind Speed mph",
	},
)

var Prom_windgust = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "windgust",
		Help: "Instant wind speed mph",
	},
)

var Prom_windDirection = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "winddirection",
		Help: "Wind Direction Deg",
	},
)

// called by prometheus
func init() {
	logger.Infof("%v: Initialize prometheus...", time.Now().Format(time.RFC822))
	prometheus.MustRegister(
		Prom_atmPresure,
		Prom_humidity,
		Prom_rainRatePerHour,
		Prom_rainDayTotal,
		Prom_temperature,
		Prom_windspeed,
		Prom_windgust,
		Prom_windDirection)
}

func main() {
	logger.Infof("Starting weather station [%v]", version)

	testMode := flag.Bool("test", false, "test mode, does not send met office data")
	flag.Parse()

	if *testMode {
		logger.Info("TEST MODE")
	}

	logger.Infof("%v: Initialize sensors...", time.Now().Format(time.RFC822))
	w := weatherstation{}
	w.testMode = *testMode

	w.s = sensors.InitSensors()
	if w.s != nil {
		logger.Error("Failed to initialise sensors")
		logger.Exit(1)
	}
	defer (*w.s.Closer).Close()

	//setup heartbeat
	heartbeatPin := gpioreg.ByName(constants.HeartbeatLed)
	if heartbeatPin == nil {
		logger.Errorf("Failed to find %v - heartbeat pin", constants.HeartbeatLed)
		// failed heartbeat LED is not critical
	}
	_ = heartbeatPin.Out(gpio.Low)
	w.HeartbeatLed = led.NewLED("Heartbeat LED", &heartbeatPin)

	w.data = data.CreateWeatherData()

	//connect to database
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		logger.Errorf("Failed to initialise database: [%v]", err)
		logger.Exit(1)
	}
	defer db.Close()

	w.dbq = postgres.New(db)

	if !(*testMode) {
		go w.MetofficeProcessor()
	}

	go w.heartbeat()

	// start web service
	http.HandleFunc("/", w.handler)
	sendData, ok := os.LookupEnv("SENDPROMDATA")
	if ok && sendData == "true" && !(*testMode) {
		logger.Info("Starting webservice...")
		http.Handle("/metrics", promhttp.Handler())
		logger.Fatal(http.ListenAndServe(":80", nil))
	} else {
		logger.Fatal(http.ListenAndServe(":80", nil))
		logger.Info("Exiting")
	}
	defer logger.Info("Exiting...")
}

func (w *weatherstation) heartbeat() {
	logger.Info("Heartbeat started")
	for {
		logger.Info("Sending heartbeat")
		w.HeartbeatLed.Flash()
		time.Sleep(time.Second * 30)
	}
}

func (w *weatherstation) handler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	pres, hum := w.s.Atm.GetHumidityAndPressure()
	wd := webdata{
		TempHiRes: w.s.Atm.GetTemperature().Float64(),
		Humidity:  hum.Float64(),
		Pressure:  pres.Float64(),
		RainHr:    w.s.Rain.GetRate().Float64(),
		RainRate:  w.s.Rain.GetMinuteRate().Float64(),
		TimeNow:   time.Now().Format(time.RFC822),
		WindDir:   w.s.Wind.GetDirection(),
		WindSpeed: w.s.Wind.GetSpeed(),
	}

	js, err := json.Marshal(wd)
	if err != nil {
		logger.Errorf("JSON error [%v]", err)
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Infof("Web read: \n[%v]", string(js))
	_, _ = rw.Write(js) // not much we can do if this fails
}
