# racing-mqtt

Go service that fetches WEC, IMSA, WRC, and Supercars schedules from the [Lightsouts API](https://lightsouts.com/) and publishes retained JSON to Mosquitto, with optional Home Assistant MQTT discovery.

No Home Assistant credentials or custom integrations required — HA only needs MQTT configured to subscribe and run automations (e.g. Telegram → Pushover).

## Topics

### WEC / IMSA / WRC / Supercars (Lightsouts)

| Topic | Contents |
|-------|----------|
| `home/racing/active/current` | Live session (or next-up if idle) with full session detail |
| `home/racing/upcoming/current` | Filtered upcoming sessions within horizon |
| `home/racing/summary` | Aggregated snapshot: `live`, `current`, `next_by_series`, `upcoming_count` |

### Formula Drift

| Topic | Contents |
|-------|----------|
| `home/racing/drifting/active/current` | Live FD round weekend (or next-up) |
| `home/racing/drifting/upcoming/current` | Upcoming FD rounds |
| `home/racing/drifting/summary` | `live`, `current`, `next`, `upcoming_count` |

Default source is [formulad.com/schedule](https://www.formulad.com/schedule). Set `DRIFT_SOURCE=ics` and `DRIFT_ICS_URL` to use an iCalendar feed instead (e.g. [driftcalendar.com](https://driftcalendar.com/)).

Drift events include YouTube links matched from the [Formula DRIFT channel](https://www.youtube.com/formuladrift): `youtube_url` (teaser or best match), `youtube_live_url` (live redirect during event weekend), and `youtube_channel_url`.

```yaml
# Heads-up before Formula Drift Las Vegas with YouTube teaser
trigger:
  - platform: mqtt
    topic: home/racing/drifting/summary
condition:
  - condition: template
    value_template: "{{ trigger.payload_json.next.slug == 'las-vegas' }}"
action:
  - service: notify.telegram
    data:
      title: "Formula Drift: {{ trigger.payload_json.next.name }}"
      message: >
        {{ trigger.payload_json.next.venue }} — {{ trigger.payload_json.next.start }}
        Watch: {{ trigger.payload_json.next.youtube_url }}
```

### Virginia International Raceway (VIR)

| Topic | Contents |
|-------|----------|
| `home/racing/vir/active/current` | Live VIR event weekend (or next-up) |
| `home/racing/vir/upcoming/current` | Upcoming VIR events |
| `home/racing/vir/summary` | `live`, `current`, `next`, `upcoming_count` |

Schedule is scraped from [virnow.com/events](https://virnow.com/events/). Each event includes `venue`, `location`, and optional `link` to the event page.

```yaml
# Notify when the next VIR event is IMSA
trigger:
  - platform: mqtt
    topic: home/racing/vir/summary
    value_template: "{{ value_json.next.series }}"
    payload: "IMSA"
action:
  - service: notify.telegram
    data:
      title: "IMSA at VIR"
      message: >
        {{ value_json.next.name }} at {{ value_json.next.venue }}
        starts {{ value_json.next.start }}
```

### Formula Drift (example automation)

```yaml
# Notify when a Formula Drift round weekend is live
trigger:
  - platform: mqtt
    topic: home/racing/drifting/summary
    value_template: "{{ value_json.live }}"
    payload: "true"
action:
  - service: notify.telegram
    data:
      title: "Formula Drift live"
      message: >
        Round {{ value_json.current.round }}: {{ value_json.current.name }}
        at {{ value_json.current.location }}
```

### Results (OpenWEC)

## Quick start

```bash
cp .env.example .env
# Edit MQTT_HOST and other settings

go test ./...
mise run racing    # loads .env
```

## Configuration

```bash
# Lightsouts API series slugs
RACING_SERIES_SLUGS=wec,imsa-sportscar-championship,wrc,supercars-championship

# Session categories: practice, qualifying, sprint, race, other
RACING_CATEGORIES=race,qualifying

# Polling (seconds); shortens to 30s within 5 minutes of session start/end
RACING_POLL_INTERVAL_SECONDS=900
RACING_UPCOMING_DAYS=90

# MQTT
MQTT_HOST=127.0.0.1
MQTT_PORT=1883
MQTT_TOPIC_PREFIX=home/racing
MQTT_CLIENT_ID=racing-mqtt
MQTT_DISCOVERY_ENABLED=true
MQTT_DISCOVERY_PREFIX=homeassistant
```

See [`.env.example`](.env.example) for all variables.

## Home Assistant automations

With MQTT discovery enabled, entities appear under **Racing MQTT**:

- `binary_sensor.racing_mqtt_live` — on while a session is live
- `sensor.racing_mqtt_next_wec` — next WEC session start (timestamp)
- `sensor.racing_mqtt_next_imsa` — next IMSA session start (timestamp)
- `sensor.racing_mqtt_next_wrc` — next WRC session start (timestamp)
- `sensor.racing_mqtt_next_supercars_championship` — next Supercars session start (timestamp)

Discovery sensors are created for each series in `RACING_SERIES_SLUGS` that has a known `next_by_series` key.

Example automation for race-start notification:

```yaml
trigger:
  - platform: mqtt
    topic: home/racing/summary
    value_template: "{{ value_json.live }}"
    payload: "true"
condition:
  - condition: template
    value_template: "{{ trigger.payload_json.current.category == 'race' }}"
action:
  - service: notify.telegram
    data:
      title: "Race live"
      message: >
        {{ trigger.payload_json.current.series }}:
        {{ trigger.payload_json.current.session }}
        at {{ trigger.payload_json.current.circuit }}
```

## Docker

```bash
docker compose up -d --build
```

Image: `ghcr.io/resnostyle/racing-mqtt:latest`

## Layout

```
cmd/racing/                 entrypoint
internal/lib/lightsouts/    Lightsouts API client
internal/lib/drift/          Formula Drift schedule (formulad + ICS)
internal/lib/vir/            VIR schedule (virnow.com)
internal/lib/ics/            iCalendar parser
internal/lib/mqttpub/       MQTT publisher + discovery
internal/lib/env/           env helpers
internal/lib/poll/          interruptible wait
internal/racing/            config, state, payloads, publish
```

## Results (OpenWEC)

After each race ends, the service matches Lightsouts sessions to [OpenWEC](https://openwec.com/) by circuit name and start time, then publishes podium data (top 3 per class). No API key required for results.

| Topic | Contents |
|-------|----------|
| `home/racing/results/wec/current` | Latest WEC race results |
| `home/racing/results/imsa/current` | Latest IMSA race results |
| `home/racing/results/{series}/{event-slug}` | Per-event retained copy |

```yaml
# Notify when WEC results are published
trigger:
  - platform: mqtt
    topic: home/racing/results/wec/current
action:
  - service: notify.telegram
    data:
      title: "WEC results"
      message: >
        {{ value_json.event }}:
        {% for class, rows in value_json.podium_by_class.items() %}
        {{ class }} P1 #{{ rows[0].car_number }} {{ rows[0].team }}
        {% endfor %}
```

Set `OPENWEC_ENABLED=false` to disable. Lap/stint analytics need a free key at [openwec.com/api-keys](https://openwec.com/api-keys).

## Roadmap

| Phase | Source | Status |
|-------|--------|--------|
| Schedules | Lightsouts API | Done |
| Results | OpenWEC API | Done |
| Formula Drift | formulad.com / ICS | Done |
| VIR | virnow.com/events | Done |

## Credits

Schedule data from [lightsouts.com](https://lightsouts.com/). Results from [OpenWEC](https://openwec.com/). Lightsouts fetch logic adapted from the [Lightsouts HA integration](https://github.com/mm98/ha-lightsouts-motorsport-calendar).
