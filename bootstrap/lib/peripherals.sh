#!/bin/sh
scan_peripherals() { PERIPHERALS=''; for _x in usb pci i2c spi gpio video4linux snd bluetooth net; do _yes=false; [ -d "$(path /sys/bus/$_x)" ] || [ -d "$(path /sys/class/$_x)" ] || [ -d "$(path /dev/$_x)" ] && _yes=true; PERIPHERALS="$PERIPHERALS$_x=$_yes;"; done; }
peripherals_json() { printf '{"availability":'; json_quote "$PERIPHERALS"; printf ',"device_accessibility":"not probed"}'; }
