#!/usr/bin/env python3
"""把 4×2 洋紅底角色圖集轉成 Modern Icon 透明 64×56 戰鬥素材。"""

import argparse
from pathlib import Path

from PIL import Image, ImageOps


def remove_magenta(image: Image.Image) -> Image.Image:
    src = image.convert("RGBA")
    pixels = []
    for red, green, blue, _ in src.getdata():
        # 純洋紅的 chroma 值最高；在邊緣保留 80 階柔化，減少粉紅鋸齒。
        chroma = min(red, blue) - green
        alpha = max(0, min(255, (210 - chroma) * 255 // 80))
        pixels.append((red, green, blue, alpha))
    src.putdata(pixels)
    return src


def fit_sprite(image: Image.Image) -> Image.Image:
    bbox = image.getbbox()
    if bbox is None:
        raise ValueError("圖格去背後是空的")
    sprite = image.crop(bbox)
    sprite.thumbnail((60, 52), Image.Resampling.LANCZOS)
    out = Image.new("RGBA", (64, 56))
    out.alpha_composite(sprite, ((64 - sprite.width) // 2, 55 - sprite.height))
    return out


def extract_sheet(path: Path, names: list[str]) -> list[Image.Image]:
    sheet = Image.open(path).convert("RGBA")
    if len(names) != 8:
        raise ValueError("4×2 圖集必須提供八個名稱")
    cell_width, cell_height = sheet.width // 4, sheet.height // 2
    out = []
    for i in range(8):
        x, y = (i % 4) * cell_width, (i // 4) * cell_height
        cell = sheet.crop((x, y, x + cell_width, y + cell_height))
        out.append(fit_sprite(remove_magenta(cell)))
    return out


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--south", type=Path, required=True)
    parser.add_argument("--north", type=Path, required=True)
    parser.add_argument("--east", type=Path, required=True)
    parser.add_argument("--names", required=True, help="八個輸出名稱，以逗號分隔")
    parser.add_argument("--out", type=Path, required=True)
    args = parser.parse_args()

    names = args.names.split(",")
    views = {
        "south": extract_sheet(args.south, names),
        "north": extract_sheet(args.north, names),
        "east": extract_sheet(args.east, names),
    }
    views["west"] = [ImageOps.mirror(image) for image in views["east"]]
    args.out.mkdir(parents=True, exist_ok=True)
    for direction, images in views.items():
        for name, image in zip(names, images):
            image.save(args.out / f"monster-{name}-{direction}.png")


if __name__ == "__main__":
    main()
