#!/usr/bin/env python3
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "public"
SOURCE = ROOT / "public" / "logo.png"
BLACK = (0, 0, 0, 255)
CLEAR = (0, 0, 0, 0)


def content_bbox(im: Image.Image) -> tuple[int, int, int, int]:
    pixels = im.load()
    w, h = im.size
    minx, miny, maxx, maxy = w, h, 0, 0
    for y in range(h):
        for x in range(w):
            r, g, b, a = pixels[x, y]
            if a > 10 and (r + g + b) > 30:
                minx, miny = min(minx, x), min(miny, y)
                maxx, maxy = max(maxx, x), max(maxy, y)
    return minx, miny, maxx + 1, maxy + 1


def fit(src: Image.Image, size: int, padding_ratio: float = 0, bg: tuple[int, int, int, int] = CLEAR) -> Image.Image:
    canvas = Image.new("RGBA", (size, size), bg)
    inner = max(1, int(size * (1 - 2 * padding_ratio)))
    scale = min(inner / src.width, inner / src.height)
    nw = max(1, int(src.width * scale))
    nh = max(1, int(src.height * scale))
    resized = src.resize((nw, nh), Image.Resampling.LANCZOS)
    canvas.paste(resized, ((size - nw) // 2, (size - nh) // 2), resized)
    return canvas


def save(img: Image.Image, name: str) -> None:
    path = OUT / name
    path.parent.mkdir(parents=True, exist_ok=True)
    img.save(path, "PNG")


def main() -> None:
    src = Image.open(SOURCE).convert("RGBA")
    glyph = src.crop(content_bbox(src))
    save(fit(src, 512), "icons/icon-512.png")
    save(fit(src, 192), "icons/icon-192.png")
    save(fit(glyph, 512, 0.22, BLACK), "icons/icon-maskable-512.png")
    save(fit(glyph, 192, 0.22, BLACK), "icons/icon-maskable-192.png")
    save(fit(src, 180), "apple-touch-icon.png")
    save(fit(src, 32), "favicon.png")


if __name__ == "__main__":
    main()
