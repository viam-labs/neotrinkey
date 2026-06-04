# boot.py — enable the USB CDC *data* serial channel so the host driver has a
# dedicated port to send pixel commands on (separate from the REPL console).
# Copy this (and code.py) onto the NeoTrinkey's CIRCUITPY drive, then reset.
import usb_cdc

usb_cdc.enable(console=True, data=True)
