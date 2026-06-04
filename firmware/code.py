# code.py — NeoTrinkey firmware for the viam:neotrinkey:trinkey driver.
#
# Reads newline-terminated lines of 12 comma-separated 0-255 integers
# ("r,g,b,r,g,b,r,g,b,r,g,b\n") from the USB CDC data port and drives the 4
# on-board NeoPixels. Pairs with go.bug.st/serial on the host side.
#
# Requires CircuitPython + the `neopixel` library, and boot.py enabling
# usb_cdc.data. Tested on the Adafruit NeoTrinkey (SAMD21).
import board
import neopixel
import usb_cdc

NUM = 4
pixels = neopixel.NeoPixel(board.NEOPIXEL, NUM, brightness=1.0, auto_write=False)
serial = usb_cdc.data

# Boot indicator: dim white, then off.
pixels.fill((8, 8, 8))
pixels.show()
pixels.fill((0, 0, 0))
pixels.show()

buf = b""
while True:
    if serial is None:
        continue
    n = serial.in_waiting
    if not n:
        continue
    buf += serial.read(n)
    while b"\n" in buf:
        line, buf = buf.split(b"\n", 1)
        parts = line.strip().split(b",")
        try:
            vals = [int(p) for p in parts]
        except ValueError:
            continue
        if len(vals) >= NUM * 3:
            for i in range(NUM):
                pixels[i] = (vals[i * 3], vals[i * 3 + 1], vals[i * 3 + 2])
            pixels.show()
