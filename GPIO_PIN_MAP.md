# Clementina GPIO MIA Pin Map

This map connects the `clementina-gpio` emulator on a Raspberry Pi 5 to a
Pico running `clementina-mia`.

Source mappings:

- Emulator GPIO lines: `pkg/common/gpio_controller.go`
- Pico MIA GPIO lines: `clementina-mia/src/mia/hardware/gpio_mapping.h`

Assumptions:

- The emulator runs against the Pi 5 default `--gpio-chip gpiochip4`.
- Pi GPIO line offsets correspond to BCM GPIO numbers.
- The Pico is powered separately over USB for first bring-up.
- The Pi and Pico share ground.

Do not connect Pi `5V`, Pi `3V3`, Pico `VBUS`, Pico `VSYS`, or Pico
`3V3_OUT` together during the first test.

## Raspberry Pi 5 Header Order

Wire the Raspberry Pi 5 40-pin header in physical pin order:

| Pi pin | Pi signal | Connect |
|---:|---|---|
| 1 | 3V3 | NC |
| 2 | 5V | NC |
| 3 | GPIO2 | Pico pin 11 / GP8 / `D0` |
| 4 | 5V | NC |
| 5 | GPIO3 | Pico pin 12 / GP9 / `D1` |
| 6 | GND | Pico GND |
| 7 | GPIO4 | Pico pin 14 / GP10 / `D2` |
| 8 | GPIO14 | NC |
| 9 | GND | Optional GND |
| 10 | GPIO15 | NC |
| 11 | GPIO17 | Pico pin 15 / GP11 / `D3` |
| 12 | GPIO18 | Pico pin 27 / GP21 / `PHI2` RED |
| 13 | GPIO27 | Pico pin 16 / GP12 / `D4` |
| 14 | GND | Optional GND |
| 15 | GPIO22 | Pico pin 17 / GP13 / `D5` |
| 16 | GPIO23 | NC |
| 17 | 3V3 | NC |
| 18 | GPIO24 | NC |
| 19 | GPIO10 | Pico pin 19 / GP14 / `D6` |
| 20 | GND | Optional GND |
| 21 | GPIO9 | Pico pin 20 / GP15 / `D7` |
| 22 | GPIO25 | NC |
| 23 | GPIO11 | Pico pin 21 / GP16 / `A0` |
| 24 | GPIO8 | NC |
| 25 | GND | Optional GND |
| 26 | GPIO7 | NC |
| 27 | GPIO0 | NC, ID pin |
| 28 | GPIO1 | NC, ID pin |
| 29 | GPIO5 | Pico pin 22 / GP17 / `A1` |
| 30 | GND | Optional GND |
| 31 | GPIO6 | Pico pin 24 / GP18 / `A2` |
| 32 | GPIO12 | Pico pin 29 / GP22 / `IRQB` GREEN |
| 33 | GPIO13 | Pico pin 25 / GP19 / `A3` |
| 34 | GND | Optional GND |
| 35 | GPIO19 | Pico pin 26 / GP20 / `A4` |
| 36 | GPIO16 | Pico pin 10 / GP7 / `R/W` YELLOW |
| 37 | GPIO26 | Pico pin 32 / GP27 / `MIA_RESETB` GREY |
| 38 | GPIO20 | Pico pin 31 / GP26 / `RESB` BLACK |
| 39 | GND | Optional GND |
| 40 | GPIO21 | Pico pin 9 / GP6 / `MIA_CS` WHITE |

## Pico Header Order

Wire the Pico header in physical pin order:

| Pico pin | Pico signal | Connect |
|---:|---|---|
| 1 | GP0 | NC |
| 2 | GP1 | NC |
| 3 | GND | Pi GND |
| 4 | GP2 | NC |
| 5 | GP3 | NC |
| 6 | GP4 | NC |
| 7 | GP5 | NC |
| 8 | GND | Optional GND |
| 9 | GP6 | Pi pin 40 / GPIO21 / `MIA_CS` |
| 10 | GP7 | Pi pin 36 / GPIO16 / `R/W` |
| 11 | GP8 | Pi pin 3 / GPIO2 / `D0` |
| 12 | GP9 | Pi pin 5 / GPIO3 / `D1` |
| 13 | GND | Optional GND |
| 14 | GP10 | Pi pin 7 / GPIO4 / `D2` |
| 15 | GP11 | Pi pin 11 / GPIO17 / `D3` |
| 16 | GP12 | Pi pin 13 / GPIO27 / `D4` |
| 17 | GP13 | Pi pin 15 / GPIO22 / `D5` |
| 18 | GND | Optional GND |
| 19 | GP14 | Pi pin 19 / GPIO10 / `D6` |
| 20 | GP15 | Pi pin 21 / GPIO9 / `D7` |
| 21 | GP16 | Pi pin 23 / GPIO11 / `A0` |
| 22 | GP17 | Pi pin 29 / GPIO5 / `A1` |
| 23 | GND | Optional GND |
| 24 | GP18 | Pi pin 31 / GPIO6 / `A2` |
| 25 | GP19 | Pi pin 33 / GPIO13 / `A3` |
| 26 | GP20 | Pi pin 35 / GPIO19 / `A4` |
| 27 | GP21 | Pi pin 12 / GPIO18 / `PHI2` |
| 28 | GND | Optional GND |
| 29 | GP22 | Pi pin 32 / GPIO12 / `IRQB` |
| 30 | RUN | NC |
| 31 | GP26 | Pi pin 38 / GPIO20 / `RESB` |
| 32 | GP27 | Pi pin 37 / GPIO26 / `MIA_RESETB` |
| 33 | AGND | NC |
| 34 | GP28 | NC |
| 35 | ADC_VREF | NC |
| 36 | 3V3_OUT | NC |
| 37 | 3V3_EN | NC |
| 38 | GND | Optional GND |
| 39 | VSYS | NC |
| 40 | VBUS | NC |

## Signal Summary

| Signal | Pi BCM | Pi pin | Pico GPIO | Pico pin | Direction |
|---|---:|---:|---:|---:|---|
| `D0` | GPIO2 | 3 | GP8 | 11 | Bidirectional |
| `D1` | GPIO3 | 5 | GP9 | 12 | Bidirectional |
| `D2` | GPIO4 | 7 | GP10 | 14 | Bidirectional |
| `D3` | GPIO17 | 11 | GP11 | 15 | Bidirectional |
| `D4` | GPIO27 | 13 | GP12 | 16 | Bidirectional |
| `D5` | GPIO22 | 15 | GP13 | 17 | Bidirectional |
| `D6` | GPIO10 | 19 | GP14 | 19 | Bidirectional |
| `D7` | GPIO9 | 21 | GP15 | 20 | Bidirectional |
| `A0` | GPIO11 | 23 | GP16 | 21 | Pi to Pico |
| `A1` | GPIO5 | 29 | GP17 | 22 | Pi to Pico |
| `A2` | GPIO6 | 31 | GP18 | 24 | Pi to Pico |
| `A3` | GPIO13 | 33 | GP19 | 25 | Pi to Pico |
| `A4` | GPIO19 | 35 | GP20 | 26 | Pi to Pico |
| `MIA_CS` | GPIO21 | 40 | GP6 | 9 | Pi to Pico |
| `R/W` | GPIO16 | 36 | GP7 | 10 | Pi to Pico |
| `PHI2` | GPIO18 | 12 | GP21 | 27 | Pico to Pi |
| `IRQB` | GPIO12 | 32 | GP22 | 29 | Pico to Pi |
| `RESB` | GPIO20 | 38 | GP26 | 31 | Pico to Pi |
| `MIA_RESETB` | GPIO26 | 37 | GP27 | 32 | Pi to Pico |

