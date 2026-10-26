import { describe, expect, test } from "vitest";

import type { FirstrunFirstRun, FirstrunStepState } from "../api/gen/model";
import { around, currentPlace, placeAt, shownSteps } from "./first-run";

const view = (...steps: FirstrunStepState[]): FirstrunFirstRun => ({ open: true, steps });

const address = { id: "address", state: "done" } as const;
const passkey = { id: "passkey", state: "todo" } as const;
const language = { id: "language", state: "skipped" } as const;
const roots = { id: "roots", state: "todo" } as const;
const targets = { id: "targets", state: "todo" } as const;

describe("shownSteps", () => {
  test("keeps the passkey step where this browser can hold a passkey", () => {
    expect(shownSteps(view(address, passkey, roots), { passkeys: true })).toEqual([
      address,
      passkey,
      roots,
    ]);
  });

  test("leaves the passkey step out where it can't", () => {
    expect(shownSteps(view(address, passkey, roots), { passkeys: false })).toEqual([
      address,
      roots,
    ]);
  });
});

describe("currentPlace", () => {
  test("is the first step still to do", () => {
    expect(currentPlace([address, language, roots, targets])).toBe("roots");
  });

  test("is the done screen once every step is set or skipped", () => {
    expect(currentPlace([address, language])).toBe("done");
  });

  test("passes a passkey step the browser hides, so the gate never lands on it", () => {
    expect(currentPlace(shownSteps(view(address, passkey, roots), { passkeys: false }))).toBe(
      "roots",
    );
  });
});

describe("placeAt", () => {
  const shown = [address, language, roots, targets];

  test("goes back to a step already set", () => {
    expect(placeAt(shown, "address")).toBe("address");
  });

  test("opens a later step the URL names", () => {
    expect(placeAt(shown, "targets")).toBe("targets");
  });

  test("holds a step that isn't shown, or isn't a step, on the current one", () => {
    expect(placeAt(shown, "passkey")).toBe("roots");
    expect(placeAt(shown, "elsewhere")).toBe("roots");
  });

  test("holds the done screen until no step is todo", () => {
    expect(placeAt(shown, "done")).toBe("roots");
    expect(placeAt([address, language], "done")).toBe("done");
  });
});

describe("around", () => {
  const shown = [address, language, roots, targets];

  test("numbers a step, names the steps ahead and links its neighbours", () => {
    expect(around(shown, "language")).toEqual({
      number: 2,
      count: 4,
      back: "address",
      next: "roots",
      ahead: ["roots", "targets"],
    });
  });

  test("has no Back on the first step", () => {
    expect(around(shown, "address").back).toBeUndefined();
  });

  test("leads from the last step to the done screen", () => {
    expect(around(shown, "targets")).toMatchObject({ next: "done", ahead: [] });
  });

  test("goes back from the done screen to the last step", () => {
    expect(around(shown, "done")).toEqual({
      number: undefined,
      count: 4,
      back: "targets",
      next: "done",
      ahead: [],
    });
  });
});
