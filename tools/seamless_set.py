#!/usr/bin/env python3
"""將一組高解析母圖縮成共用邊界的可交錯鋪設 PNG。"""

import argparse
from pathlib import Path

from PIL import Image


def mix(a, b, t):
    return tuple(round(a[i] * (1.0 - t) + b[i] * t) for i in range(3))


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--width", type=int, required=True)
    p.add_argument("--height", type=int, required=True)
    p.add_argument("--blend", type=int, required=True)
    p.add_argument("pairs", nargs="+", help="來源:輸出")
    args = p.parse_args()
    if len(args.pairs) < 2:
        p.error("至少需要兩張同組素材")
    if args.blend < 2 or args.blend * 2 >= min(args.width, args.height):
        p.error("blend 必須 >=2 且小於短邊一半")

    images = []
    outputs = []
    for spec in args.pairs:
        src, sep, out = spec.partition(":")
        if not sep:
            p.error(f"缺少來源:輸出分隔：{spec}")
        im = Image.open(src).convert("RGB")
        images.append(im.resize((args.width, args.height), Image.Resampling.LANCZOS))
        outputs.append(Path(out))

    # 同組所有圖的左右邊收斂到同一條 target；blend 內側逐步回到各自原圖。
    horizontal = []
    for y in range(args.height):
        samples = []
        for im in images:
            samples.extend((im.getpixel((0, y)), im.getpixel((args.width - 1, y))))
        horizontal.append(tuple(round(sum(c[i] for c in samples) / len(samples)) for i in range(3)))
    for im in images:
        original = im.copy()
        for y in range(args.height):
            target = horizontal[y]
            for d in range(args.blend):
                t = d / (args.blend - 1)
                im.putpixel((d, y), mix(target, original.getpixel((d, y)), t))
                x = args.width - 1 - d
                im.putpixel((x, y), mix(target, original.getpixel((x, y)), t))

    # 上下邊同理；target 從已處理左右邊的結果計算，四個角會一起閉合。
    vertical = []
    for x in range(args.width):
        samples = []
        for im in images:
            samples.extend((im.getpixel((x, 0)), im.getpixel((x, args.height - 1))))
        vertical.append(tuple(round(sum(c[i] for c in samples) / len(samples)) for i in range(3)))
    for im, out in zip(images, outputs):
        original = im.copy()
        for x in range(args.width):
            target = vertical[x]
            for d in range(args.blend):
                t = d / (args.blend - 1)
                im.putpixel((x, d), mix(target, original.getpixel((x, d)), t))
                y = args.height - 1 - d
                im.putpixel((x, y), mix(target, original.getpixel((x, y)), t))
        out.parent.mkdir(parents=True, exist_ok=True)
        im.save(out)


if __name__ == "__main__":
    main()
