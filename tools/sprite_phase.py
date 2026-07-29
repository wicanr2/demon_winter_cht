#!/usr/bin/env python3
"""由透明 A 幀產生乾淨的 1 px 呼吸／浮動 B 幀。"""

import argparse
from pathlib import Path

from PIL import Image


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("sources", nargs="+", type=Path)
    parser.add_argument("--suffix", default="-b")
    args = parser.parse_args()

    for path in args.sources:
        src = Image.open(path).convert("RGBA")
        out = Image.new("RGBA", src.size)
        # 整體上浮一像素，不縮放、不旋轉，避免小尺寸透明邊緣產生接縫。
        out.alpha_composite(src, (0, -1))
        target = path.with_name(path.stem + args.suffix + path.suffix)
        out.save(target)


if __name__ == "__main__":
    main()
