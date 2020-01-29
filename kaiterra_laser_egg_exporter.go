package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	addr       = flag.String("addr", ":8000", "http address to listen on")
	apiKey     = flag.String("api_key", "", "API key for Kaiterra")
	deviceUUID = flag.String("device_uuid", "", "UUID for Device")

	pmDesc = prometheus.NewDesc(
		"kaiterra_particulate_matter",
		"PM2.5 or PM10 (µg/m³), post-calibration",
		[]string{"microns"},
		nil,
	)
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
		kc,
	)

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	http.ListenAndServe(*addr, nil)
}

type kaiterraCollector struct {
	c *http.Client
}

func (kc kaiterraCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(kc, ch)
}

func (kc kaiterraCollector) Collect(ch chan<- prometheus.Metric) {
	req, err := http.NewRequest("GET", "https://api.kaiterra.cn/v1/lasereggs/", nil)
	if err != nil {
		panic(err)
	}
	req.URL.Path += *deviceUUID
	q := url.Values{}
	q.Add("key", *apiKey)
	req.URL.RawQuery = q.Encode()
	log.Printf(req.URL.String())
	resp, err := kc.c.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	decoded := JSONResponse{}
	json.NewDecoder(resp.Body).Decode(&decoded)
	log.Printf("%+v", decoded)
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
		return
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
}

type JSONResponse struct {
	// 128-bit UUID, Device ID for Laser Egg
	ID  string
	AQI JSONAQI `json:"info.aqi"`
}

type JSONAQI struct {
	// RFC3339 (a refinement of ISO8601), which looks like 2016-12-07T05:32:16Z
	TS   string
	Data JSONData
}

// https://www.kaiterra.com/dev/#header-pollutant-data
type JSONData struct {
	// Relative humidity in % (0-100)
	Humidity float64
	// PM10 (µg/m³), post-calibration
	PM10 float64
	// PM2.5 (µg/m³), post-calibration
	PM25 float64
	// Temperature in Celsius
	Temp float64
}
