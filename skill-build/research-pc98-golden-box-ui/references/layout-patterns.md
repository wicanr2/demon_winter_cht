# PC-98 Golden Box CJK UI reference

## Contents

1. Reference corpus
2. Stable cross-title patterns
3. Typography measurements
4. Screen-type comparison
5. Color and hierarchy
6. Application checklist
7. Limits and copyright boundary

## 1. Reference corpus

The most direct Dragonlance examples are:

| Common Chinese name | Original title | PC-98 release context |
|---|---|---|
| 克萊恩英豪 | *Champions of Krynn* | Opera House port; Pony Canyon, 1992 |
| 幽靈騎士 | *Death Knights of Krynn* | Opera House port; Pony Canyon, 1993 |

Cross-check them against four Forgotten Realms games so one port batch is not mistaken
for a universal rule:

- *Pool of Radiance*
- *Curse of the Azure Bonds*
- *Secret of the Silver Blades*
- *Pools of Darkness*

Public galleries:

- [Champions of Krynn overview](https://retroarchives.fr/champions-of-krynn/)
- [Champions of Krynn PC-98 screenshots](https://www.mobygames.com/game/833/champions-of-krynn/screenshots/pc98/)
- [Death Knights of Krynn overview](https://retroarchives.fr/death-knights-of-krynn/)
- [Death Knights of Krynn PC-98 screenshots](https://www.mobygames.com/game/2219/death-knights-of-krynn/screenshots/pc98/)
- [Pool of Radiance PC-98 screenshots](https://www.mobygames.com/game/502/pool-of-radiance/screenshots/pc98/)
- [Curse of the Azure Bonds PC-98 screenshots](https://www.mobygames.com/game/503/curse-of-the-azure-bonds/screenshots/pc98/)
- [Secret of the Silver Blades PC-98 screenshots](https://www.mobygames.com/game/504/secret-of-the-silver-blades/screenshots/pc98/)
- [Pools of Darkness PC-98 screenshots](https://www.mobygames.com/game/505/pools-of-darkness/screenshots/pc98/)

Verify live pages when freshness, exact counts, release metadata, or quotations matter.
Do not redistribute gallery screenshots in a repository without permission.

## 2. Stable cross-title patterns

The corpus consistently uses a 640×400 logical canvas and reorganizes it into:

1. A large primary content region for a map, illustration, battlefield, or text.
2. A stable context/status region for party or current-target information.
3. A separate command or system-prompt region, usually at the bottom.

The robust lesson is information hierarchy, not ornamental frame geometry:

- persistent values remain in predictable places;
- narrative text and actionable commands occupy distinct regions;
- combat reserves the battlefield for spatial reasoning;
- current actor and target details stay in a side panel;
- long prose may use the full content width;
- high-density screens switch to aligned tables;
- different modes share hierarchy but need not share identical window proportions.

## 3. Typography measurements

Typical native-image measurements:

| Property | PC-98 Golden Box observation |
|---|---:|
| Logical canvas | 640×400 |
| CJK cell | approximately 16×16 |
| Wide narrative measure | approximately 608 px / 38 full-width cells |
| Dense table line height | approximately 16 px |
| Narrative line height | often approximately 16 px |

These measurements support a 16×15 bitmap glyph placed in a 16×16 cell. They do not
require a Traditional Chinese remake to use 16 px narrative leading. Dense Traditional
Chinese strokes, synthetic bolding, and punctuation may justify 18–20 px leading.

Measure native pixels:

- distinguish glyph bitmap size from advance width and line height;
- record both full-width CJK cells and proportional punctuation advances;
- test rare, dense Traditional Chinese characters;
- keep scaling integral and filtering disabled;
- check title, prose, tables, numbers, mixed Latin/CJK text, and disabled states.

## 4. Screen-type comparison

Compare like with like:

| Target screen | Golden Box evidence to inspect |
|---|---|
| Exploration | viewport, party block, message area, command row |
| Dialogue | prose width, portrait/illustration allocation, paging prompt |
| Combat | battlefield size, current actor, target stats, report and commands |
| Camp | action groups, spell/item lists, selection and confirmation |
| Shop | tabular alignment, price columns, inventory capacity |
| Character creation | form labels, values, help text, focus highlight |
| Automap | map dominance, legend, position/orientation indicator |

Do not infer dialogue capacity from a combat panel or a shop table from narrative prose.

## 5. Color and hierarchy

Observed ports commonly distinguish content/commands, selection/system prompts, and
headings with separate colors on black. The reusable concept is a semantic color role:

- primary text and ordinary values;
- current selection and input prompt;
- section heading;
- warning, negative value, or enemy;
- disabled, expressed by a pattern as well as reduced brightness.

Keep these roles constant across EGA, CGA, PC-98-inspired, and modernized themes. A theme
switch changes color and ornament assets, not wrapping, geometry, selection, or state.

## 6. Application checklist

At 640×400 native resolution verify:

1. No half glyph, clipped frame, or text overflow at the maximum intended line width.
2. Five consecutive lines of dense Traditional Chinese remain distinct.
3. Exploration, dungeon, combat, and sea combat clearly separate primary content,
   persistent status, contextual message, and commands.
4. The combat side panel prioritizes current actor, target, HP/SP, protection, and weapon.
5. Narrative and current commands cannot be confused.
6. EGA/CGA/modern themes preserve semantic color roles.
7. Theme switching does not change line breaks, column widths, active selection, or state.
8. Screenshots are captured without interpolation for parity review.

Suggested audit table:

| Dimension | Reference observation | Target measurement | Decision | Confidence |
|---|---|---|---|---|
| Canvas | 640×400 | … | retain/change | measured/inferred |
| CJK cell | ~16×16 | … | retain/change | measured/inferred |
| Narrative width | ~38 cells | … | retain/change | measured/inferred |
| Regions | content/status/commands | … | retain/change | cross-title/project |

## 7. Limits and copyright boundary

Safe to adopt:

- region hierarchy;
- grid and alignment principles;
- semantic color roles;
- CJK capacity and pagination lessons;
- native-resolution validation methods.

Do not copy:

- jeweled borders, Dragonlance marks, logos, portraits, sprites, illustrations;
- game-specific wording, icons, maps, or decorative compositions;
- PC-98 colors as if they proved DOS EGA/CGA authenticity;
- Golden Box commands or mechanics when the target game differs.

The target game's executable, manual, and original assets remain the behavioral and
historical oracle. PC-98 Golden Box ports are comparative localization evidence only.
