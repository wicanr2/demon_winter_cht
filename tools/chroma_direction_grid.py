#!/usr/bin/env python3
"""把「四方向 × 多個外觀」洋紅底圖集轉成透明 64×56 runtime 素材。"""

import argparse
from pathlib import Path

from PIL import Image

from chroma_spritesheet import fit_sprite, remove_magenta


DIRECTIONS = ("north", "east", "south", "west")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--sheet", type=Path, required=True)
    parser.add_argument("--names", required=True, help="每列的輸出名稱，以逗號分隔")
    parser.add_argument("--prefix", required=True)
    parser.add_argument("--out", type=Path, required=True)
    args = parser.parse_args()

    names = [name.strip() for name in args.names.split(",") if name.strip()]
    if not names:
        raise ValueError("至少要提供一個列名")
    sheet = Image.open(args.sheet).convert("RGBA")
    args.out.mkdir(parents=True, exist_ok=True)
    for row, name in enumerate(names):
        for col, direction in enumerate(DIRECTIONS):
            cell = sheet.crop((
                col * sheet.width // 4,
                row * sheet.height // len(names),
                (col + 1) * sheet.width // 4,
                (row + 1) * sheet.height // len(names),
            ))
            sprite = fit_sprite(remove_magenta(cell))
            sprite.save(args.out / f"{args.prefix}-{name}-{direction}.png")


if __name__ == "__main__":
    main()
