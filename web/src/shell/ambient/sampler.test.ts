import { expect, test } from "vitest";

import { artwork } from "../../test/ambient/fixtures";
import { ceiling } from "../../theme.gen";
import { compositeLuminance, sampleHeight, sampleImage, sampleWidth } from "./sampler";

test.each([
  { name: "white", hue: 220, saturation: 12, luminance: 1, brightness: 0.255 },
  { name: "black", hue: 220, saturation: 12, luminance: 0, brightness: 1 },
  { name: "red", hue: 0, saturation: 70, luminance: 0.2126, brightness: 0.536 },
  { name: "two-tone", hue: 221, saturation: 53, luminance: 1, brightness: 0.255 },
] as const)(
  "the $name fixture samples to $hue / $saturation% at brightness $brightness",
  async (expected) => {
    const { name, ...reading } = expected;
    const sample = await sampleImage(artwork[name]);
    expect({ ...sample, luminance: Math.round(sample.luminance * 10_000) / 10_000 }).toEqual(
      reading,
    );
  },
);

test("the brightness factor is the largest thousandth that holds the ceiling", () => {
  const white = new Uint8ClampedArray(sampleWidth * sampleHeight * 4).fill(255);
  const tint = { hue: 220, saturation: 12 };
  expect(compositeLuminance(white, tint, 0.255)).toBeLessThanOrEqual(ceiling);
  expect(compositeLuminance(white, tint, 0.256)).toBeGreaterThan(ceiling);
});
