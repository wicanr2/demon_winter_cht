#!/usr/bin/env python3
"""
parse_town.py — Demon's Winter TOWN*.DAT / EXITS.DAT / ITEMLOC*.DAT 解析器

已驗證的格式(詳見 docs/formats/town-and-map.md):
    TOWN*.DAT (512 bytes) = 30 筆 17-byte 定長記錄, 無 header, 尾端多 2 bytes。
        每筆記錄: byte0 = 類型碼(facility/entity type code), byte1-13 = payload(13 bytes,
        多為 0, 有時是價格/數量等數值), byte14-16 = 尾端 marker(常見 "0a 01 01",
        但非固定值, 目前語意未完全解出)。
    EXITS.DAT (330 bytes) = 假設為 165 筆 (X,Y) 座標對 (2 bytes/筆), 未含 0x00,
        數值上限 64, 與 64x64 地圖座標範圍吻合(信心中等, 未跨資料交叉驗證)。
    ITEMLOCB.DAT / ITEMLOCX.DAT (256 bytes, 兩檔逐 byte 相同) = 85 筆 (X, Y, map_id)
        三元組 (3 bytes/筆) + 1 byte 尾端 0xff。前段是有效資料(map_id 出現 1/3/4/5,
        與現存 MAP1/MAP3/MAP5 吻合, 並證實 map_id=4 在遊戲邏輯上存在), 後段是
        0xff 填充的空位, 中間一小段(約 record 50-54)是無法辨識的殘餘資料。

用法:
    python3 tools/parse_town.py town <TOWN*.DAT路徑>
        印出 30 筆記錄的 hex dump + 欄位切分

    python3 tools/parse_town.py town-all <DEM_DATA目錄> [--town-txt TOWN.TXT路徑]
        掃描 TOWN1..25.DAT, 印出每座城鎮的城名(來自 TOWN.TXT)+ 記錄摘要對照表

    python3 tools/parse_town.py exits <EXITS.DAT路徑>
        印出 EXITS.DAT 依 2-byte stride 切分的內容

    python3 tools/parse_town.py itemloc <ITEMLOCB或X.DAT路徑>
        印出 ITEMLOC*.DAT 依 3-byte (X,Y,map_id) stride 切分的內容

只用標準庫, 不需要額外套件。
"""
import sys
import os
import glob
import re
import argparse

TOWN_RECORD_SIZE = 17
TOWN_RECORD_COUNT = 30
TOWN_FILE_SIZE = 512


def load_town_names(town_txt_path):
    if not os.path.exists(town_txt_path):
        return []
    with open(town_txt_path, 'rb') as f:
        data = f.read()
    parts = data.split(b'\x00')
    # 前 25 個字串是城鎮名(已驗證, 順序對應 TOWN1..25.DAT), 之後是劇情字串
    names = [p.decode('ascii', 'replace') for p in parts[:25]]
    return names


def cmd_town(args):
    with open(args.path, 'rb') as f:
        data = f.read()
    if len(data) != TOWN_FILE_SIZE:
        print(f'[警告] 檔案長度 {len(data)} 不等於預期的 {TOWN_FILE_SIZE}', file=sys.stderr)
    print(f'{args.path}  ({len(data)} bytes)')
    for n in range(TOWN_RECORD_COUNT):
        off = n * TOWN_RECORD_SIZE
        rec = data[off:off + TOWN_RECORD_SIZE]
        if len(rec) < TOWN_RECORD_SIZE:
            break
        code = rec[0]
        payload = rec[1:14]
        tail = rec[14:17]
        nonzero_payload = any(b != 0 for b in payload)
        marker = ''
        if code == 0xff:
            marker = ' [FF-sentinel/空位]'
        print(f'  rec{n:2d} @0x{off:03x}  code=0x{code:02x}({code:3d})  '
              f'payload={payload.hex(" ")}  tail={tail.hex(" ")}'
              f'{"  <-- 有非零 payload" if nonzero_payload else ""}{marker}')
    tail2 = data[510:512]
    print(f'  尾端 2 bytes (offset 510-511): {tail2.hex(" ")}')


def cmd_town_all(args):
    town_dir = args.dem_data_dir
    town_txt = args.town_txt or os.path.join(town_dir, 'TOWN.TXT')
    names = load_town_names(town_txt)

    files = sorted(
        glob.glob(os.path.join(town_dir, 'TOWN*.DAT')),
        key=lambda f: int(re.search(r'TOWN(\d+)\.DAT', os.path.basename(f)).group(1))
    )
    print(f'{"編號":>4} {"城名":<16} {"檔案":<14} {"有效記錄數(非FF)":>16} {"type-A(純flag)":>14} {"type-B(有payload)":>17} {"FF空位":>8}')
    for f in files:
        idx = int(re.search(r'TOWN(\d+)\.DAT', os.path.basename(f)).group(1))
        name = names[idx - 1] if 0 < idx <= len(names) else '?'
        data = open(f, 'rb').read()
        typeA = typeB = ff = 0
        for n in range(TOWN_RECORD_COUNT):
            off = n * TOWN_RECORD_SIZE
            rec = data[off:off + TOWN_RECORD_SIZE]
            if len(rec) < TOWN_RECORD_SIZE:
                break
            code = rec[0]
            payload = rec[1:14]
            tail = rec[14:17]
            if code == 0xff:
                ff += 1
            elif all(b == 0 for b in payload) and tail == b'\x0a\x01\x01':
                typeA += 1
            elif any(b != 0 for b in payload):
                typeB += 1
        valid = typeA + typeB
        print(f'{idx:>4} {name:<16} {os.path.basename(f):<14} {valid:>16} {typeA:>14} {typeB:>17} {ff:>8}')


def cmd_exits(args):
    with open(args.path, 'rb') as f:
        data = f.read()
    print(f'{args.path}  ({len(data)} bytes)  假設: 2-byte (X,Y) 座標對, 中信心度')
    n = len(data) // 2
    for i in range(n):
        x, y = data[2 * i], data[2 * i + 1]
        print(f'  #{i:3d}  x={x:3d}  y={y:3d}')
    if len(data) % 2:
        print('  剩餘 byte:', data[n * 2:].hex(' '))


def cmd_itemloc(args):
    with open(args.path, 'rb') as f:
        data = f.read()
    print(f'{args.path}  ({len(data)} bytes)  假設: 3-byte (X, Y, map_id) 三元組, 已驗證')
    n = len(data) // 3
    for i in range(n):
        x, y, mid = data[3 * i], data[3 * i + 1], data[3 * i + 2]
        tag = ''
        if (x, y, mid) == (0xff, 0xff, 0xff):
            tag = ' [空位]'
        elif (x, y) == (0, 0):
            tag = ' [x=y=0, 疑似未使用]'
        elif mid not in (1, 3, 4, 5) and mid != 0xff:
            tag = ' [map_id 超出已知範圍 1/3/4/5, 疑似殘餘資料]'
        print(f'  #{i:3d}  x={x:3d}  y={y:3d}  map_id={mid:3d}{tag}')
    if len(data) % 3:
        print('  剩餘 byte:', data[n * 3:].hex(' '))


def main():
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = p.add_subparsers(dest='cmd', required=True)

    pt = sub.add_parser('town', help='解析單一 TOWN*.DAT')
    pt.add_argument('path')
    pt.set_defaults(func=cmd_town)

    pta = sub.add_parser('town-all', help='掃描整個 DEM_DATA 目錄下的 TOWN*.DAT, 印摘要對照表')
    pta.add_argument('dem_data_dir')
    pta.add_argument('--town-txt')
    pta.set_defaults(func=cmd_town_all)

    pe = sub.add_parser('exits', help='解析 EXITS.DAT')
    pe.add_argument('path')
    pe.set_defaults(func=cmd_exits)

    pi = sub.add_parser('itemloc', help='解析 ITEMLOCB.DAT / ITEMLOCX.DAT')
    pi.add_argument('path')
    pi.set_defaults(func=cmd_itemloc)

    args = p.parse_args()
    args.func(args)


if __name__ == '__main__':
    main()
