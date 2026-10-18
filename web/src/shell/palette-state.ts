import { appleViewer } from "../ui/platform";
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

export const paletteShortcut = appleViewer
  ? { label: "⌘K", keys: "Meta+K" }
  : { label: "Ctrl K", keys: "Control+K" };

export const isPaletteShortcut = (event: KeyboardEvent): boolean =>
  event.key.toLowerCase() === "k" &&
  !event.altKey &&
  !event.shiftKey &&
  (appleViewer ? event.metaKey && !event.ctrlKey : event.ctrlKey && !event.metaKey);
