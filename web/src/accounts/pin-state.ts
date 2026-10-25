import type { IdentityPinState } from "../api/gen/model";
import { clock } from "../format";

type PinState = IdentityPinState["state"];

export const pinLabel: { readonly [S in PinState]: string } = {
  fine: "Fine",
  waiting: "Waiting",
  off: "Turned off",
};

const sentences: { readonly [S in PinState]: (until: string | undefined) => string } = {
  fine: () => "The PIN works.",
  waiting: (until) =>
    until === undefined
      ? "Sign-in is paused after too many wrong PINs."
      : `Sign-in is paused until ${clock(until)} after too many wrong PINs.`,
  off: () =>
    "The PIN was turned off after too many wrong PINs. Send a Reset link to set a new one.",
};

export const pinSentence = (pin: IdentityPinState): string => sentences[pin.state](pin.until);
