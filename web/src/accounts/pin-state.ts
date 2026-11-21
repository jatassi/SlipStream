import type { IdentityPinState } from "../api/gen/model";
import type { Clock } from "../ui/clock";
import { whenWords } from "../ui/time-words";

type PinState = IdentityPinState["state"];

export const pinLabel: { readonly [S in PinState]: string } = {
  fine: "Fine",
  waiting: "Waiting",
  off: "Locked",
};

const sentences: {
  readonly [S in PinState]: (until: string | undefined, clock: Clock) => string;
} = {
  fine: () => "The PIN works.",
  waiting: (until, clock) =>
    until === undefined
      ? "After too many wrong PINs, sign-in is paused."
      : `After too many wrong PINs, sign-in is paused ${whenWords(until, "until", clock)}.`,
  off: () => "The PIN is locked after too many wrong PINs. Send a Reset link to set a new one.",
};

export const pinSentence = (pin: IdentityPinState, clock: Clock): string =>
  sentences[pin.state](pin.until, clock);
