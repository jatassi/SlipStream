import { createFlag } from "./flag";

const palette = createFlag();

export const openPalette = () => {
  palette.set(true);
};

export const closePalette = () => {
  palette.set(false);
};

export const usePaletteOpen = palette.useValue;

const apple = /Mac|iPhone|iPad|iPod/u.test(navigator.userAgent);

export const paletteShortcut = apple
  ? { label: "⌘K", keys: "Meta+K" }
  : { label: "Ctrl K", keys: "Control+K" };
