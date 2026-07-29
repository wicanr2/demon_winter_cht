#!/usr/bin/env python3
"""把多欄、正常／冬季兩列的 terrain 母稿切成 64×56 不透明 PNG。"""

import argparse
from pathlib import Path

from PIL import Image


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--sheet", type=Path, required=True)
    parser.add_argument("--indices", required=True, help="由左至右的十六進位索引")
    parser.add_argument("--out", type=Path, required=True)
    args = parser.parse_args()

    indices = [int(value.strip(), 16) for value in args.indices.split(",") if value.strip()]
    if not indices:
        raise ValueError("至少要有一個索引")
    sheet = Image.open(args.sheet).convert("RGB")
    args.out.mkdir(parents=True, exist_ok=True)
    for row, season in enumerate(("normal", "winter")):
        for col, index in enumerate(indices):
            cell = sheet.crop((
                col * sheet.width // len(indices),
                row * sheet.height // 2,
                (col + 1) * sheet.width // len(indices),
                (row + 1) * sheet.height // 2,
            ))
            cell = cell.resize((64, 56), Image.Resampling.LANCZOS)
            cell.save(args.out / f"{season}-special-{index:02x}.png")


if __name__ == "__main__":
    main()
