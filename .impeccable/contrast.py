#!/usr/bin/env python3
"""Prove DESIGN.md's colour tokens hold WCAG 2.2 AA over every tint the ambient can take.

The ambient sampler emits a hue (0-360) and a saturation clamped to 18-70%; the neutral
ambient is 214 / 14%. Translucent surfaces composite over the ambient in sRGB, as browsers do.
The script finds the brightest ambient (relative luminance) every pair survives, which
DESIGN.md records as the ambient ceiling, then checks every pair at that ceiling.

Run: python3 .impeccable/contrast.py
"""
import colorsys
import sys

HUES = range(0, 360, 3)
SATS = [14, 18, 30, 45, 60, 70]


def hsl(h, s, l, a=1.0):
    r, g, b = colorsys.hls_to_rgb(h / 360, l / 100, s / 100)
    return (r * 255, g * 255, b * 255, a)


def hexa(code, a=1.0):
    code = code.lstrip('#')
    return (int(code[0:2], 16), int(code[2:4], 16), int(code[4:6], 16), a)


def over(top, bottom):
    a = top[3]
    return tuple(top[i] * a + bottom[i] * (1 - a) for i in range(3)) + (1.0,)


def lum(c):
    def ch(v):
        v /= 255
        return v / 12.92 if v <= 0.04045 else ((v + 0.055) / 1.055) ** 2.4
    return 0.2126 * ch(c[0]) + 0.7152 * ch(c[1]) + 0.0722 * ch(c[2])


def ratio(fg, bg):
    a, b = sorted((lum(fg), lum(bg)), reverse=True)
    return (a + 0.05) / (b + 0.05)


def ambient(h, s, y):
    lo, hi = 0.0, 100.0
    for _ in range(40):
        mid = (lo + hi) / 2
        if lum(hsl(h, min(100, s * 1.3), mid)) < y:
            lo = mid
        else:
            hi = mid
    return hsl(h, min(100, s * 1.3), lo)


def tokens(h, s):
    return {
        'bg': hsl(h, s * .5, 5),
        'glass': hsl(h, s * .55, 8, .56),
        'glass-2': hsl(h, s * .6, 13, .6),
        'chrome': hsl(h, s * .5, 6, .64),
        'pop': hsl(h, s * .55, 12, .84),
        'sheet': hsl(h, s * .4, 10, .88),
        'fill': hsl(h, 40, 92, .1),
        'fill-2': hsl(h, 40, 92, .16),
        'seg-on': hsl(h, 30, 92, .22),
        'live-chip': hsl(h, 50, 26, .7),
        'ink-1': hsl(h, 25, 96),
        'ink-2': hsl(h, 18, 83),
        'ink-3': hsl(h, 14, 71),
        'ink-4': hsl(h, 12, 60),
        'solid': hexa('f6f4f0'),
        'on-solid': hexa('111111'),
        'fail': hexa('ff6b61'),
        'fail-ink': hexa('ffb0a9'),
        'fail-deep': hexa('2a0603'),
        'fail-card': hsl(3, 48, 11, .97),
        'fail-text': hexa('fff3f1'),
        'fail-dim': hexa('ffe2df', .74),
        'warn': hexa('ffb547'),
        'warn-deep': hexa('2b1a00'),
        'warn-card': hsl(37, 50, 9, .97),
        'warn-text': hexa('fff6e9'),
        'warn-dim': hexa('ffeed4', .76),
        'ok': hexa('5fd394'),
        'ok-deep': hexa('03230f'),
    }


# (ink, surface stack from top to bottom over the ambient, minimum ratio, what it is)
PAIRS = [
    ('ink-1', [], 4.5, 'primary text on bare ambient'),
    ('ink-2', [], 4.5, 'secondary text on bare ambient'),
    ('ink-3', [], 4.5, 'tertiary text on bare ambient'),
    ('ink-4', [], 3.0, 'separators and idle marks on bare ambient (non-text)'),
    ('ink-3', ['glass'], 4.5, 'tertiary text on glass'),
    ('ink-3', ['chrome'], 4.5, 'tertiary text on chrome'),
    ('ink-3', ['glass-2', 'glass'], 4.5, 'tertiary text on raised glass'),
    ('ink-3', ['fill', 'chrome'], 4.5, 'placeholder in a top-bar field'),
    ('ink-2', ['fill', 'glass'], 4.5, 'ghost button text on glass'),
    ('ink-1', ['fill-2', 'glass'], 4.5, 'button text, hovered, on glass'),
    ('ink-1', ['seg-on', 'fill', 'glass'], 4.5, 'selected segment'),
    ('ink-2', ['fill', 'glass'], 4.5, 'unselected segment'),
    ('ink-3', ['pop'], 4.5, 'menu meta on popover'),
    ('ink-3', ['sheet'], 4.5, 'sheet subtitle'),
    ('ink-1', ['live-chip', 'glass'], 4.5, 'live chip label'),
    ('fail-ink', [], 4.5, 'failure link in the status sentence'),
    ('warn', [], 4.5, 'warning link in the status sentence'),
    ('fail-ink', ['glass'], 4.5, 'failure ink on glass'),
    ('fail-text', ['fail-card'], 4.5, 'text on a failure card'),
    ('fail-dim', ['fail-card'], 4.5, 'dim text on a failure card'),
    ('fail-ink', ['fail-card'], 4.5, 'labels on a failure card'),
    ('warn-text', ['warn-card'], 4.5, 'text on a warning line'),
    ('warn-dim', ['warn-card'], 4.5, 'dim text on a warning line'),
    ('warn', ['warn-card'], 4.5, 'amber label on a warning line'),
    ('fail-deep', ['fail'], 4.5, 'ink on a failure chip or button'),
    ('warn-deep', ['warn'], 4.5, 'ink on a warning chip or button'),
    ('ok-deep', ['ok'], 4.5, 'ink on a healthy chip'),
    ('on-solid', ['solid'], 4.5, 'ink on solid white (primary, Met)'),
    ('fail', ['glass'], 3.0, 'failure dot or icon on glass (non-text)'),
    ('warn', ['glass'], 3.0, 'warning dot or icon on glass (non-text)'),
    ('ok', ['glass'], 3.0, 'healthy dot or icon on glass (non-text)'),
    ('ok', [], 3.0, 'healthy dot or icon on bare ambient (non-text)'),
]


def worst(y):
    out = {}
    for h in HUES:
        for s in SATS:
            t = tokens(h, s)
            amb = ambient(h, s, y)
            for ink, stack, need, label in PAIRS:
                bg = amb
                for layer in reversed(stack):
                    bg = over(t[layer], bg)
                fg = t[ink] if t[ink][3] == 1 else over(t[ink], bg)
                r = ratio(fg, bg)
                key = (ink, tuple(stack), label)
                if key not in out or r < out[key][0]:
                    out[key] = (r, need, h, s)
    return out


def passes(y):
    return all(r >= need for r, need, _, _ in worst(y).values())


def main():
    if not passes(0.0):
        print('fails even on a black ambient')
    lo, hi = 0.0, 0.5
    for _ in range(30):
        mid = (lo + hi) / 2
        lo, hi = (mid, hi) if passes(mid) else (lo, mid)
    ceiling = round(lo - 0.0005, 3)
    print(f'ambient ceiling (relative luminance): {ceiling}')
    res = worst(ceiling)
    failed = False
    for (ink, stack, label), (r, need, h, s) in sorted(res.items(), key=lambda kv: kv[1][0] / kv[1][1]):
        mark = 'ok  ' if r >= need else 'FAIL'
        failed |= r < need
        print(f'{mark} {r:5.2f} >= {need:.1f}  {label:52s} worst at h{h} s{s}')
    sys.exit(1 if failed else 0)


if __name__ == '__main__':
    main()
