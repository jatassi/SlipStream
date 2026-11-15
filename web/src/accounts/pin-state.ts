import type { IdentityPinState } from "../api/gen/model";
import { clock } from "../format";

type PinState = IdentityPinState["state"];

export const pinLabel: { readonly [S in PinState]: string } = {
  fine: "Fine",
  waiting: "Waiting",
  off: "Locked",
};

const sentences: { readonly [S in PinState]: (until: string | undefined) => string } = {
  fine: () => "The PIN works.",
  waiting: (until) =>
    until === undefined
      ? "After too many wrong PINs, sign-in is paused."
      : `After too many wrong PINs, sign-in is paused until ${clock(until)}.`,
  off: () => "The PIN is locked after too many wrong PINs. Send a Reset link to set a new one.",
};

export const pinSentence = (pin: IdentityPinState): string => sentences[pin.state](pin.until);
