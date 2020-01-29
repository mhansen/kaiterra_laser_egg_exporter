# Kaiterra Laser Egg Exporter

A prometheus.io exporter for Kaiterra Laser Egg Air Quality / Particulate Matter Sensors.

Pushed to Docker Hub at https://hub.docker.com/r/markhnsn/kaiterra_laser_egg_exporter

Get an API Key at https://dashboard.kaiterra.cn/.

Find your device UUID in the Kaiterra app.

Example usage:

    $ go build
    $ ./kaiterra_laser_egg_exporter --addr=:8000 --device_uuid=3d6d04a2-ba9f-11e6-9598-0800200c9a66 --api_key=kOpAgVMnz2zM5l6XKQwv4JmUEvopnmUewFKXQ0Wvf9Su72a9

    $ curl http://localhost:8000/metrics
    ...
    # HELP kaiterra_humidity relative humidity in % (0-100)
    # TYPE kaiterra_humidity gauge
    kaiterra_humidity 60.46
    # HELP kaiterra_particulate_matter PM2.5 or PM10 (µg/m³), post-calibration
    # TYPE kaiterra_particulate_matter gauge
    kaiterra_particulate_matter{microns="10"} 16
    kaiterra_particulate_matter{microns="2.5"} 15
    # HELP kaiterra_temperature_celsius temperature in Celsius
    # TYPE kaiterra_temperature_celsius gauge
    kaiterra_temperature_celsius 26.74
    # HELP kaiterra_timestamp_seconds Timestamp was measured at. Unix seconds.
    # TYPE kaiterra_timestamp_seconds gauge
    kaiterra_timestamp_seconds 1.580291945e+09

