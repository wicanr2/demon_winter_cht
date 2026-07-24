#!/usr/bin/env python3
"""
parse_map.py — Demon's Winter MAP*.MAP / SUM.MAP 解析器

用法:
    python3 tools/parse_map.py ascii <MAP檔路徑> [--out 輸出檔]
        把 4097-byte MAP*.MAP 印成 64x64 ASCII 圖 (1 byte header + 64*64 tile array)

    python3 tools/parse_map.py stats <MAP檔路徑>
        印出 tile 值分佈統計 (驗證假設用)

    python3 tools/parse_map.py entropy <檔案路徑>
        對任意檔案做位元組熵值分析,判斷是否像壓縮資料

    python3 tools/parse_map.py route <MAP檔路徑> --start X,Y --dirs "4E 3N 3E 5S..."
        依方位字串 (N/S/E/W, 前綴數字=格數) 走一遍路線, 印出沿途 tile,
        用來跟攻略文字比對驗證

只用標準庫, 不需要額外套件。
"""
import sys
import math
import argparse
import collections

MAP_W = 64
MAP_H = 64
MAP_SIZE = 1 + MAP_W * MAP_H  # 4097

# tile 值 -> ASCII 字元的猜測對照表(視覺化用,不代表確定語意)
# 依出現頻率高的當「地板/牆」等常見地形,細節見 docs/formats/town-and-map.md
TILE_CHARS = {
    0: ' ',
    13: '.',
    98: '#',
    35: '+',
    42: '%',
    19: '~',
    86: '"',
    17: '-',
    49: '=',
    18: ':',
    51: '*',
    90: '^',
    88: 'o',
}


def load_map(path):
    with open(path, 'rb') as f:
        data = f.read()
    if len(data) != MAP_SIZE:
        print(f'[警告] {path} 長度 {len(data)} 不等於預期的 {MAP_SIZE} (1 + 64*64)', file=sys.stderr)
    header = data[0] if data else None
    body = data[1:1 + MAP_W * MAP_H]
    grid = [body[y * MAP_W:(y + 1) * MAP_W] for y in range(MAP_H)]
    return header, grid


def cmd_ascii(args):
    header, grid = load_map(args.path)
    lines = [f'# {args.path}  header_byte=0x{header:02x}' if header is not None else f'# {args.path}']
    for row in grid:
        line = ''.join(TILE_CHARS.get(b, '?') for b in row)
        lines.append(line)
    out = '\n'.join(lines)
    if args.out:
        with open(args.out, 'w') as f:
            f.write(out + '\n')
        print(f'寫入 {args.out} ({len(grid)} 列)')
    else:
        print(out)


def cmd_stats(args):
    header, grid = load_map(args.path)
    flat = b''.join(bytes(row) for row in grid)
    c = collections.Counter(flat)
    print(f'{args.path}  header=0x{header:02x}  distinct_tiles={len(c)}')
    for val, cnt in c.most_common(40):
        pct = cnt / len(flat) * 100
        print(f'  tile {val:3d} (0x{val:02x})  {cnt:5d}  {pct:5.1f}%  char={TILE_CHARS.get(val, "?")}')


def shannon_entropy(data):
    if not data:
        return 0.0
    c = collections.Counter(data)
    n = len(data)
    ent = 0.0
    for cnt in c.values():
        p = cnt / n
        ent -= p * math.log2(p)
    return ent


def cmd_entropy(args):
    with open(args.path, 'rb') as f:
        data = f.read()
    ent = shannon_entropy(data)
    c = collections.Counter(data)
    print(f'{args.path}')
    print(f'  檔案大小: {len(data)} bytes')
    print(f'  distinct byte 值: {len(c)} / 256')
    print(f'  Shannon entropy: {ent:.3f} bits/byte  (隨機/壓縮資料理論上限 8.0, 純文字/重複結構通常 <6)')
    print(f'  最常見的 10 個 byte 值:')
    for val, cnt in c.most_common(10):
        print(f'    0x{val:02x}  {cnt:6d}  {cnt/len(data)*100:5.1f}%')
    # 判斷建議
    if ent > 7.5:
        verdict = '高熵,像壓縮或加密資料'
    elif ent > 6.0:
        verdict = '中高熵,可能是壓縮資料或密集的二進位結構(不一定是壓縮)'
    else:
        verdict = '中低熵,像有結構的原始資料(tile array / 表格),不太像壓縮'
    print(f'  初步判斷: {verdict} (僅供參考,不是確定結論)')


DIR_VEC = {
    'N': (0, -1),
    'S': (0, 1),
    'E': (1, 0),
    'W': (-1, 0),
}


def parse_route(route_str):
    """把 '4E 3N 3E 5S 3E 3S 5W S 2W 2N 2W N 6W' 解析成 [(dx,dy,count), ...]"""
    steps = []
    for tok in route_str.split():
        tok = tok.strip()
        if not tok:
            continue
        i = 0
        while i < len(tok) and tok[i].isdigit():
            i += 1
        count = int(tok[:i]) if i > 0 else 1
        d = tok[i:].upper()
        if d not in DIR_VEC:
            print(f'[警告] 無法解析的方位 token: {tok!r}', file=sys.stderr)
            continue
        steps.append((d, count))
    return steps


def cmd_route(args):
    header, grid = load_map(args.path)
    x, y = map(int, args.start.split(','))
    steps = parse_route(args.dirs)
    print(f'{args.path}  header=0x{header:02x}  起點=({x},{y})')

    def tile_at(cx, cy):
        if 0 <= cx < MAP_W and 0 <= cy < MAP_H:
            return grid[cy][cx]
        return None

    t0 = tile_at(x, y)
    print(f'  起點 tile = {t0} (0x{t0:02x})' if t0 is not None else '  起點超出邊界!')
    for d, count in steps:
        dx, dy = DIR_VEC[d]
        for _ in range(count):
            x += dx
            y += dy
            t = tile_at(x, y)
            if t is None:
                print(f'  {d} -> ({x},{y}) 超出邊界 (64x64)!')
                continue
            print(f'  {d} -> ({x},{y})  tile={t:3d} (0x{t:02x}) char={TILE_CHARS.get(t, "?")}')
    print('  終點:', (x, y))


def main():
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = p.add_subparsers(dest='cmd', required=True)

    pa = sub.add_parser('ascii', help='印成 64x64 ASCII 圖')
    pa.add_argument('path')
    pa.add_argument('--out')
    pa.set_defaults(func=cmd_ascii)

    ps = sub.add_parser('stats', help='tile 值分佈統計')
    ps.add_argument('path')
    ps.set_defaults(func=cmd_stats)

    pe = sub.add_parser('entropy', help='位元組熵值分析(判斷是否壓縮)')
    pe.add_argument('path')
    pe.set_defaults(func=cmd_entropy)

    pr = sub.add_parser('route', help='依方位字串走路線,印沿途 tile')
    pr.add_argument('path')
    pr.add_argument('--start', required=True, help='起點座標 X,Y')
    pr.add_argument('--dirs', required=True, help='方位字串, 例如 "4E 3N 3E 5S"')
    pr.set_defaults(func=cmd_route)

    args = p.parse_args()
    args.func(args)


if __name__ == '__main__':
    main()
