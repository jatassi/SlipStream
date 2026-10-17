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
      ? "Too many wrong PINs. Sign-in waits before the next try."
      : `Too many wrong PINs. Sign-in waits until ${clock(until)}.`,
  off: () => "Too many wrong PINs turned the PIN off. Send a Reset link to set a new one.",
};

export const pinSentence = (pin: IdentityPinState): string => sentences[pin.state](pin.until);
