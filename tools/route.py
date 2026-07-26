#!/usr/bin/env python3
"""在地城地圖上找一條路，輸出 `tools/playthrough.sh` 吃得下的按鍵腳本。

**這不是作弊捷徑。** 它只做路線規劃 —— 隊伍照樣一步一步走過去，
每一格照樣觸發它該觸發的東西。A4 的規則是「不用 debug 捷徑走完主線」
（`docs/re/64` §3），查地圖找路跟玩家拿張攻略地圖是同一件事。

會做這支是因為手算不可行：地城是 64×64 的迷宮，
上一次試玩用直線路徑往東走三步就撞牆了。

可通行性：`FILES.DAT` 偏移 `0x040` 起 104 bytes，索引是 `tile & 0x7f`，
值 `0xff` ＝ 牆（`docs/re/22` §2）。

用法：
    tools/route.py MAP1.MAP 9 32 16 36          # 從 (9,32) 走到 (16,36)
    tools/route.py MAP1.MAP 9 32 16 36 --check  # 只檢查目標可不可達
"""

import argparse
import sys
from collections import deque
from pathlib import Path

DATA_DIR = Path("workplace/orig/demwin/DEM_DATA")
WIDTH = HEIGHT = 64

# `FILES.DAT` 裡可通行表的位置與長度。
PASS_OFFSET = 0x040
PASS_LEN = 104
WALL = 0xFF

# 方向 → (dx, dy, xdotool 鍵名)。Y 向下為正（螢幕座標）。
DIRS = [(0, -1, "Up"), (0, 1, "Down"), (-1, 0, "Left"), (1, 0, "Right")]


def load_passability(data_dir: Path) -> bytes:
    raw = (data_dir / "FILES.DAT").read_bytes()
    return raw[PASS_OFFSET : PASS_OFFSET + PASS_LEN]


def load_tiles(path: Path) -> bytes:
    raw = path.read_bytes()
    # 4097 bytes = **1 byte header** + 64×64 tile 陣列，tile 從偏移 1 起算
    # （`internal/assets/world/mapfile.go`、`docs/formats/town-and-map.md` §2.1）。
    #
    # ⚠ 第一版寫成從偏移 0 起算，整張圖錯開一格。症狀不是「明顯壞掉」——
    # 找出來的路大致合理，只是偶爾穿牆或撞到空氣。
    # 抓到它的方法是**拿引擎實測的撞牆點當對照**：先跑一次試玩，
    # 記下軌跡裡「前方無法通行」的兩個座標，再要求這支工具同意。
    # 錯開一格時 (12,32) 會被算成可走，與實測不符 —— 一格就露餡。
    return raw[1 : 1 + WIDTH * HEIGHT]


def walkable(tiles: bytes, table: bytes, x: int, y: int) -> bool:
    if not (0 <= x < WIDTH and 0 <= y < HEIGHT):
        return False
    t = tiles[y * WIDTH + x] & 0x7F
    if t >= len(table):
        # 有效範圍是 tile 0–100（`docs/re/22` §2）。超出就當不可通行，
        # 不要照抄記憶體裡的相鄰位元組。
        return False
    return table[t] != WALL


def find_path(tiles, table, start, goal):
    """BFS。回傳座標串列（含起點終點），走不到回 None。"""
    if start == goal:
        return [start]
    prev = {start: None}
    q = deque([start])
    while q:
        cur = q.popleft()
        for dx, dy, _ in DIRS:
            nxt = (cur[0] + dx, cur[1] + dy)
            if nxt in prev or not walkable(tiles, table, *nxt):
                continue
            prev[nxt] = cur
            if nxt == goal:
                path = [nxt]
                while prev[path[-1]] is not None:
                    path.append(prev[path[-1]])
                return path[::-1]
            q.append(nxt)
    return None


def to_script(path) -> list[str]:
    """把座標串列壓成 `rep N 方向` 的腳本行。"""
    out = []
    run_key, run_n = None, 0
    for a, b in zip(path, path[1:]):
        dx, dy = b[0] - a[0], b[1] - a[1]
        key = next(k for ddx, ddy, k in DIRS if (ddx, ddy) == (dx, dy))
        if key == run_key:
            run_n += 1
            continue
        if run_key:
            out.append(f"rep {run_n} {run_key}")
        run_key, run_n = key, 1
    if run_key:
        out.append(f"rep {run_n} {run_key}")
    return out


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("mapfile")
    ap.add_argument("coords", nargs=4, type=int, metavar=("X0", "Y0", "X1", "Y1"))
    ap.add_argument("--data", default=str(DATA_DIR))
    ap.add_argument("--check", action="store_true", help="只回報可不可達與步數")
    args = ap.parse_args()

    data_dir = Path(args.data)
    p = Path(args.mapfile)
    if not p.exists():
        p = data_dir / args.mapfile
    tiles = load_tiles(p)
    table = load_passability(data_dir)

    x0, y0, x1, y1 = args.coords
    if not walkable(tiles, table, x0, y0):
        print(f"!! 起點 ({x0},{y0}) 本身就不可通行", file=sys.stderr)
    path = find_path(tiles, table, (x0, y0), (x1, y1))
    if path is None:
        print(f"!! ({x0},{y0}) 走不到 ({x1},{y1})", file=sys.stderr)
        sys.exit(1)

    if args.check:
        print(f"可達，{len(path) - 1} 步")
        return
    print(f"# {p.name}：({x0},{y0}) → ({x1},{y1})，{len(path) - 1} 步")
    for line in to_script(path):
        print(line)


if __name__ == "__main__":
    main()
