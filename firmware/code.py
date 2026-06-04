# code.py — NeoTrinkey firmware for the viam:neotrinkey:trinkey driver.
#
# Reads newline-terminated lines of 12 comma-separated 0-255 integers
# ("r,g,b,r,g,b,r,g,b,r,g,b\n") from the USB CDC data port and drives the 4
# on-board NeoPixels. Uses only built-in CircuitPython modules (no library
# bundle needed). Requires boot.py to enable usb_cdc.data.
import board
import digitalio
import neopixel_write
import usb_cdc

NUM = 4
pin = digitalio.DigitalInOut(board.NEOPIXEL)
pin.direction = digitalio.Direction.OUTPUT
serial = usb_cdc.data


def show(rgb):
    # rgb: flat list [r0,g0,b0, r1,g1,b1, ...]; NeoPixels take GRB order.
    buf = bytearray(NUM * 3)
    for i in range(NUM):
        buf[i * 3] = rgb[i * 3 + 1]      # G
        buf[i * 3 + 1] = rgb[i * 3]      # R
        buf[i * 3 + 2] = rgb[i * 3 + 2]  # B
    neopixel_write.neopixel_write(pin, buf)


# Boot indicator: dim white, then off.
show([8, 8, 8] * NUM)
show([0, 0, 0] * NUM)

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
            show(vals[: NUM * 3])
