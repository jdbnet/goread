#!/usr/bin/env python3
from pathlib import Path

from PIL import Image, ImageDraw

OUT = Path(__file__).resolve().parents[1] / "public"


def book_icon(size: int, padding_ratio: float) -> Image.Image:
    img = Image.new("RGBA", (size, size), (28, 25, 23, 255))
    draw = ImageDraw.Draw(img)
    pad = int(size * padding_ratio)
    x0, y0 = pad, pad
    x1, y1 = size - pad, size - pad
    w, h = x1 - x0, y1 - y0
    radius = max(8, int(w * 0.1))
    spine = max(6, int(w * 0.18))
    stroke = max(3, size // 42)

    draw.rounded_rectangle([x0, y0, x1, y1], radius=radius, fill="#d97706")
    draw.rounded_rectangle([x0, y0, x0 + spine + radius, y1], radius=radius, fill="#92400e")
    draw.rectangle([x0 + spine, y0, x0 + spine + radius, y1], fill="#d97706")

    page_left = x0 + spine + int(w * 0.12)
    page_right = x1 - int(w * 0.14)
    for i in range(3):
        py = y0 + int(h * (0.3 + i * 0.16))
        draw.line([(page_left, py), (page_right, py)], fill="#fde68a", width=stroke)

    return img


def save(img: Image.Image, name: str, size: int) -> None:
    path = OUT / name
    path.parent.mkdir(parents=True, exist_ok=True)
    img.resize((size, size), Image.Resampling.LANCZOS).save(path, "PNG")


def main() -> None:
    any_icon = book_icon(512, 0.14)
    maskable = book_icon(512, 0.22)
    save(any_icon, "icons/icon-512.png", 512)
    save(any_icon, "icons/icon-192.png", 192)
    save(maskable, "icons/icon-maskable-512.png", 512)
    save(maskable, "icons/icon-maskable-192.png", 192)
    save(any_icon, "apple-touch-icon.png", 180)
    save(any_icon, "favicon.png", 32)


if __name__ == "__main__":
    main()
