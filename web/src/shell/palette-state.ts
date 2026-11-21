import { shortcut } from "../ui/kbd";
import { createFlag } from "./flag";

const palette = createFlag();

export const openPalette = () => {
  palette.set(true);
};

export const closePalette = () => {
  palette.set(false);
};

export const togglePalette = () => {
  palette.set(!palette.get());
};

export const usePaletteOpen = palette.useValue;

export const paletteShortcut = shortcut(["Mod", "K"]);
