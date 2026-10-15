import { createFlag } from "./flag";

declare global {
  interface Navigator {
    readonly userAgentData?: { readonly platform: string };
  }
}

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

const apple = /mac|iphone|ipad|ipod/iu.test(
  navigator.userAgentData?.platform ?? navigator.platform,
);

export const paletteShortcut = apple
  ? { label: "⌘K", keys: "Meta+K" }
  : { label: "Ctrl K", keys: "Control+K" };

export const isPaletteShortcut = (event: KeyboardEvent): boolean =>
  event.key.toLowerCase() === "k" &&
  !event.altKey &&
  !event.shiftKey &&
  (apple ? event.metaKey && !event.ctrlKey : event.ctrlKey && !event.metaKey);
