export const sampleWidth = 24;
export const sampleHeight = 36;

/** The colour a Work lends the page: `--h` in degrees and `--s` in percent. */
export type Tint = { readonly hue: number; readonly saturation: number };

/** DESIGN.md's ambient.neutral, the registered initial values of `--h` and `--s`. */
export const neutralTint: Tint = { hue: 214, saturation: 14 };

/** DESIGN.md's ambient.ceiling: the brightest relative luminance the ambient may composite to. */
export const ceiling = 0.053;

const chromaFloor = 0.04;
const colourless: Tint = { hue: 220, saturation: 12 };

/**
 * What the sampler reads from a 24 × 36 sample of an image.
 *
 * The brightest region is the brightest single cell of the sample after the
 * backdrop's `saturate(1.8)`. A cell already averages its patch of the image,
 * and the backdrop's `blur(64px)` spans more than a cell on screen, so a
 * blurred point is an average of cells and no brighter than the brightest.
 */
export type Sample = Tint & {
  /** Relative luminance of the brightest region. */
  readonly luminance: number;
  /** The backdrop's `brightness()` factor that holds the composite at or below the ceiling. */
  readonly brightness: number;
};

type Rgb = readonly [number, number, number];
type Point = readonly [number, number];

/**
 * DESIGN.md's tint wash, in the viewport's own fractions so it holds at
 * every aspect ratio: two glows at the Work's hue, each fading to transparent
 * at 70% of its radii, over a darkening that grows towards the bottom.
 */
export const wash = {
  glows: [
    { lightness: 26, alpha: 0.55, centre: [0.3, 0.62], radii: [0.6, 0.7] },
    { lightness: 22, alpha: 0.45, centre: [0.72, 0.34], radii: [0.55, 0.6] },
  ],
  fade: 0.7,
  darkening: { saturation: 20, top: 5, bottom: 4, alpha: 0.9 },
} as const;

export const backdropFilter = { blur: 64, saturate: 1.8 } as const;

/** The tint DESIGN.md's sampler takes from RGBA pixels. */
function tintOf(pixels: Uint8ClampedArray): Tint {
  let x = 0;
  let y = 0;
  let weights = 0;
  let saturations = 0;
  for (let i = 0; i < pixels.length; i += 4) {
    const { h, s, l, c } = hsl(rgbAt(pixels, i));
    if (c < chromaFloor) {
      continue;
    }
    const w = s * s * (1 - Math.abs(l - 0.5) * 1.6);
    const radians = (h * Math.PI) / 180;
    x += w * Math.cos(radians);
    y += w * Math.sin(radians);
    weights += w;
    saturations += w * s;
  }
  if (weights === 0) {
    return colourless;
  }
  const hue = Math.round((Math.atan2(y, x) * 180) / Math.PI);
  return {
    hue: (hue + 360) % 360,
    saturation: Math.round(Math.min(70, Math.max(18, (saturations / weights) * 80))),
  };
}

export function sample(pixels: Uint8ClampedArray): Sample {
  const tint = tintOf(pixels);
  const cells = backdropCells(pixels);
  const luminance = Math.max(...cells.map((cell) => relativeLuminance(cell)));
  return { ...tint, luminance, brightness: brightnessFor(cells, tint) };
}

/**
 * The brightest relative luminance the ambient composites to with the
 * backdrop at `brightness`, taking any cell under any point of the wash.
 */
export function compositeLuminance(pixels: Uint8ClampedArray, tint: Tint, brightness: number) {
  return composite(frontier(backdropCells(pixels)), washAt(tint), brightness);
}

export async function sampleImage(url: string): Promise<Sample> {
  const image = new Image();
  image.src = url;
  await image.decode();
  const context = sampleContext();
  context.drawImage(image, 0, 0, sampleWidth, sampleHeight);
  return sample(context.getImageData(0, 0, sampleWidth, sampleHeight).data);
}

function sampleContext() {
  const canvas =
    "OffscreenCanvas" in globalThis
      ? new OffscreenCanvas(sampleWidth, sampleHeight)
      : Object.assign(document.createElement("canvas"), {
          width: sampleWidth,
          height: sampleHeight,
        });
  const context = canvas.getContext("2d", { willReadFrequently: true });
  if (context === null) {
    throw new Error("the ambient sampler has no 2D canvas");
  }
  return context;
}

function brightnessFor(cells: readonly Rgb[], tint: Tint) {
  const points = washAt(tint);
  const candidates = frontier(cells);
  let lo = 0;
  let hi = 1000;
  while (lo < hi) {
    const mid = Math.ceil((lo + hi) / 2);
    if (composite(candidates, points, mid / 1000) <= ceiling) {
      lo = mid;
    } else {
      hi = mid - 1;
    }
  }
  return lo / 1000;
}

/**
 * Over a point of the wash, the ambient is `through × backdrop + light` per
 * channel, so the wash reduces to these two terms at each point.
 */
type WashPoint = { readonly through: number; readonly light: Rgb };

// A grid at 1/24 of the viewport plus each glow's centre, where its alpha peaks.
function washAt({ hue, saturation }: Tint): WashPoint[] {
  const steps = 24;
  const points: Point[] = wash.glows.map((glow) => glow.centre);
  for (let i = 0; i <= steps; i++) {
    for (let j = 0; j <= steps; j++) {
      points.push([i / steps, j / steps]);
    }
  }
  const { darkening } = wash;
  const dark = rgbOf(hue, darkening.saturation, darkening.bottom);
  const glows = wash.glows.map((glow) => ({
    ...glow,
    colour: rgbOf(hue, saturation, glow.lightness),
  }));
  return points.map(([u, v]) => {
    let point: WashPoint = { through: 1, light: [0, 0, 0] };
    point = over(point, dark, darkening.alpha * v);
    for (const glow of glows.toReversed()) {
      const r = Math.hypot(
        (u - glow.centre[0]) / glow.radii[0],
        (v - glow.centre[1]) / glow.radii[1],
      );
      point = over(point, glow.colour, glow.alpha * Math.max(0, 1 - r / wash.fade));
    }
    return point;
  });
}

function over({ through, light }: WashPoint, colour: Rgb, alpha: number): WashPoint {
  return {
    through: through * (1 - alpha),
    light: [
      colour[0] * alpha + light[0] * (1 - alpha),
      colour[1] * alpha + light[1] * (1 - alpha),
      colour[2] * alpha + light[2] * (1 - alpha),
    ],
  };
}

function composite(cells: readonly Rgb[], points: readonly WashPoint[], brightness: number) {
  let brightest = 0;
  for (const { through, light } of points) {
    const k = through * brightness;
    for (const cell of cells) {
      brightest = Math.max(
        brightest,
        relativeLuminance([k * cell[0] + light[0], k * cell[1] + light[1], k * cell[2] + light[2]]),
      );
    }
  }
  return brightest;
}

// The composite only grows with each channel, so a cell another cell matches
// or beats in every channel can never be the brightest.
function frontier(cells: readonly Rgb[]): Rgb[] {
  return cells.filter(
    (cell, i) =>
      !cells.some(
        (other, j) =>
          other[0] >= cell[0] &&
          other[1] >= cell[1] &&
          other[2] >= cell[2] &&
          (j < i || other[0] > cell[0] || other[1] > cell[1] || other[2] > cell[2]),
      ),
  );
}

function backdropCells(pixels: Uint8ClampedArray): Rgb[] {
  const cells: Rgb[] = [];
  for (let i = 0; i < pixels.length; i += 4) {
    cells.push(saturate(rgbAt(pixels, i), backdropFilter.saturate));
  }
  return cells;
}

function rgbAt(pixels: Uint8ClampedArray, i: number): Rgb {
  return [(pixels[i] ?? 0) / 255, (pixels[i + 1] ?? 0) / 255, (pixels[i + 2] ?? 0) / 255];
}

function hsl([r, g, b]: Rgb) {
  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  const c = max - min;
  const l = (max + min) / 2;
  const s = c === 0 ? 0 : c / (1 - Math.abs(2 * l - 1));
  let h = 0;
  if (c > 0) {
    if (max === r) {
      h = 60 * (((g - b) / c + 6) % 6);
    } else if (max === g) {
      h = 60 * ((b - r) / c + 2);
    } else {
      h = 60 * ((r - g) / c + 4);
    }
  }
  return { h, s, l, c };
}

function rgbOf(hue: number, saturation: number, lightness: number): Rgb {
  const s = saturation / 100;
  const l = lightness / 100;
  const a = s * Math.min(l, 1 - l);
  const channel = (n: number) => {
    const k = (n + hue / 30) % 12;
    return l - a * Math.max(-1, Math.min(k - 3, 9 - k, 1));
  };
  return [channel(0), channel(8), channel(4)];
}

// The Filter Effects saturate() matrix, which browsers apply to sRGB values.
function saturate([r, g, b]: Rgb, s: number): Rgb {
  return [
    clamp((0.213 + 0.787 * s) * r + (0.715 - 0.715 * s) * g + (0.072 - 0.072 * s) * b),
    clamp((0.213 - 0.213 * s) * r + (0.715 + 0.285 * s) * g + (0.072 - 0.072 * s) * b),
    clamp((0.213 - 0.213 * s) * r + (0.715 - 0.715 * s) * g + (0.072 + 0.928 * s) * b),
  ];
}

function clamp(v: number) {
  return Math.min(1, Math.max(0, v));
}

function linear(v: number) {
  return v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
}

function relativeLuminance([r, g, b]: Rgb) {
  return 0.2126 * linear(r) + 0.7152 * linear(g) + 0.0722 * linear(b);
}
