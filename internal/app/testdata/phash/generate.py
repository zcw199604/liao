"""Regenerate with Pillow==11.3.0 imagehash==4.3.1 scipy==1.13.1.

Run this script from any directory. Go tests consume golden.json without Python.
Patterns are also constructed in image_hash_python_test.go; no private photos or DB.
"""
import base64
import json
from io import BytesIO
from pathlib import Path

import imagehash
from PIL import Image

cases = [
    ("rgb-large", "RGB", 1024, 768, "noise"),
    ("rgb-medium", "RGB", 256, 193, "noise"),
    ("rgb-small", "RGB", 13, 19, "noise"),
    ("rgb-no-resize", "RGB", 32, 32, "noise"),
    ("rgba", "RGBA", 79, 53, "noise"),
    ("gray", "L", 83, 61, "noise"),
    ("gray16", "I;16", 47, 61, "noise"),
    ("width-unchanged", "RGB", 32, 87, "noise"),
    ("height-unchanged", "RGB", 91, 32, "noise"),
    ("one-column", "RGB", 1, 57, "noise"),
    ("one-row", "RGB", 63, 1, "noise"),
    ("constant", "RGB", 45, 39, "constant"),
    ("gradient", "RGB", 64, 64, "gradient"),
    ("checkerboard", "RGB", 64, 64, "checkerboard"),
]
results = []
for name, mode, width, height, pattern in cases:
    img = Image.new(mode, (width, height))
    pixels = []
    for y in range(height):
        for x in range(width):
            n = ((y * width + x + 1) * 2654435761) & 0xffffffff
            n = ((n ^ (n >> 16)) * 2246822519) & 0xffffffff
            r, g, b, a = n & 255, (n >> 8) & 255, (n >> 16) & 255, (n >> 24) & 255
            if pattern == "constant":
                r, g, b = 70, 130, 190
            elif pattern == "gradient":
                r, g, b = x * 3, y * 3, (x + y) * 2
            elif pattern == "checkerboard":
                r = g = b = 255 if (x // 8 + y // 8) % 2 else 0
            pixels.append((n & 1023) if mode == "I;16" else r if mode == "L" else (r, g, b, a) if mode == "RGBA" else (r, g, b))
    if mode == "I;16":
        img = Image.frombytes(mode, (width, height), b"".join(v.to_bytes(2, "little") for v in pixels))
    else:
        img.putdata(pixels)
    gray = img.convert("L").resize((32, 32), Image.Resampling.LANCZOS)
    results.append(dict(name=name, mode=mode, width=width, height=height, pattern=pattern,
                        gray=base64.b64encode(gray.tobytes()).decode(), phash=str(imagehash.phash(img))))
    if name == "rgba":
        # Exercise real decoder inputs too. Lossy decoders may round pixels
        # differently, so only the final hash is compared for JPEG/WebP.
        for format in ("JPEG", "WEBP", "GIF"):
            encoded = BytesIO()
            img.convert("RGB").save(encoded, format=format)
            decoded = Image.open(BytesIO(encoded.getvalue()))
            results.append(dict(name=format.lower(), encoded=base64.b64encode(encoded.getvalue()).decode(),
                                phash=str(imagehash.phash(decoded)), maxDistance=2 if format == "JPEG" else 0))
            if format == "JPEG":
                # Isolate hash compatibility from libjpeg vs Go decoder rounding
                # and chroma upsampling: the exact same decoded RGB must match.
                decoded_pixels = BytesIO()
                decoded.save(decoded_pixels, format="PNG")
                results.append(dict(name="jpeg-decoded-rgb", encoded=base64.b64encode(decoded_pixels.getvalue()).decode(),
                                    gray=base64.b64encode(decoded.convert("L").resize((32, 32), Image.Resampling.LANCZOS).tobytes()).decode(),
                                    phash=str(imagehash.phash(decoded))))
Path(__file__).with_name("golden.json").write_text(json.dumps(results, indent=2) + "\n")
