package main

import (
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// 9660 registered on https://github.com/prometheus/prometheus/wiki/Default-port-allocations
	addr       = flag.String("addr", ":9660", "http address to listen on")
	apiKey     = flag.String("api_key", "", "API key for Kaiterra")
	deviceUUID = flag.String("device_uuid", "", "UUID for Device")

	pmDesc = prometheus.NewDesc(
		"kaiterra_particulate_matter",
		"PM2.5 or PM10 (µg/m³), post-calibration",
		[]string{"microns"},
		nil,
	)

	index = template.Must(template.New("index").Parse(
		`<!doctype html>
<title>Kaiterra Laser Egg Exporter</title>
<h1>Kaiterra Laser Egg Exporter</h1>
<a href="/metrics">Metrics</a>`))
)

func main() {
	flag.Parse()
	if *apiKey == "" {
		log.Fatalf("--api_key flag required")
	}
	if *deviceUUID == "" {
		log.Fatalf("--device_uuid flag required")
	}
	log.Printf("Kaiterra Laser Egg Prometheus Exporter starting on addr %s", *addr)

	reg := prometheus.NewPedanticRegistry()
	c := &http.Client{}
	kc := kaiterraCollector{c: c}
	reg.MustRegister(
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
		prometheus.NewGoCollector(),
		prometheus.NewBuildInfoCollector(),
		kc,
	)

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		err := index.Execute(w, nil)
		if err != nil {
			log.Println(err)
		}
	})
	http.ListenAndServe(*addr, nil)
}

type kaiterraCollector struct {
	c *http.Client
}

func (kc kaiterraCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(kc, ch)
}

func (kc kaiterraCollector) Collect(ch chan<- prometheus.Metric) {
	req, err := http.NewRequest("GET", "https://api.kaiterra.com/v1/lasereggs/", nil)
	if err != nil {
		panic(err)
	}
	req.URL.Path += *deviceUUID
	q := url.Values{}
	q.Add("key", *apiKey)
	req.URL.RawQuery = q.Encode()
	resp, err := kc.c.Do(req)
	if err != nil {
		log.Printf("request to %v failed: %v", req.URL.String(), err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("got non-200 code: %v, %v", resp.StatusCode, resp.Status)
		return
	}

	decoded := JSONResponse{}
	err = json.NewDecoder(resp.Body).Decode(&decoded)
	if err != nil {
		log.Printf("couldn't parse json: %v", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(
		pmDesc,
		prometheus.GaugeValue,
		float64(decoded.AQI.Data.PM10),
		"10",
	)
	ch <- prometheus.MustNewConstMetric(
		pmDesc,
		prometheus.GaugeValue,
		float64(decoded.AQI.Data.PM25),
		"2.5",
	)
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			"kaiterra_temperature_celsius",
			"temperature in Celsius",
			[]string{},
			nil,
		),
		prometheus.GaugeValue,
		float64(decoded.AQI.Data.Temp),
	)
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			"kaiterra_humidity",
			"relative humidity in % (0-100)",
			[]string{},
			nil,
		),
		prometheus.GaugeValue,
		float64(decoded.AQI.Data.Humidity),
	)
	t, err := time.Parse(time.RFC3339, decoded.AQI.TS)
	if err != nil {
		log.Printf("Couldn't parse date: %v", decoded.AQI.TS)
		// It's fine, don't return, let's just move on and output t=0
		// so we always have the same set of metrics.
	}
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			"kaiterra_timestamp_seconds",
			"Timestamp was measured at. Unix seconds.",
			[]string{},
			nil,
		),
		prometheus.GaugeValue,
		float64(t.Unix()),
	)
	Tvoc := float64(decoded.AQI.Data.Tvoc)
	if Tvoc != 0 {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				"kaiterra_total_volatile_organic_compounds_ppb",
				"Total Volatile Organic Compounds (TVOC) in ppb",
				[]string{},
				nil,
			),
			prometheus.GaugeValue,
			float64(decoded.AQI.Data.Tvoc),
		)
	}

}

// JSONResponse is the root JSON response from Kaiterra API.
// https://www.kaiterra.com/dev/#overview
type JSONResponse struct {
	// 128-bit UUID, Device ID for Laser Egg
	ID  string
	AQI JSONAQI `json:"info.aqi"`
}

// JSONAQI is timestamped pollutant data.
type JSONAQI struct {
	// RFC3339 (a refinement of ISO8601), which looks like 2016-12-07T05:32:16Z
	TS   string
	Data JSONPollutantData
}

// JSONPollutantData isData on various pollutants or other metrics (like
// temperature and humidity).
// https://www.kaiterra.com/dev/#header-pollutant-data
type JSONPollutantData struct {
	// Relative humidity in % (0-100)
	Humidity float64
	// PM10 (µg/m³), post-calibration
	PM10 float64
	// PM2.5 (µg/m³), post-calibration
	PM25 float64
	// Temperature in Celsius
	Temp float64
	// TVOC in ppb
	Tvoc float64  `json:"st03.rtvoc"`
}
