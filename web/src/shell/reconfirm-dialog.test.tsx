import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { updateIdentitySettings } from "../api/gen/query/identity/identity";
import { getGetIdentitySettingsResponseMock } from "../api/gen/query/identity/identity.msw";
import {
  ReconfirmSessionBody,
  UpdateIdentitySettingsBody,
} from "../api/gen/zod/identity/identity.zod";
import { clock } from "../format";
import { problem } from "../test/handlers";
import { renderApp } from "../test/render-app";
import { asPhone } from "../test/viewport";

const settings = getGetIdentitySettingsResponseMock();

type Patch = ReturnType<typeof UpdateIdentitySettingsBody.parse>;

const withValue = <F extends { value: number; default: number }>(
  field: F,
  value: number | null | undefined,
): F =>
  value === undefined
    ? field
    : { ...field, value: value ?? field.default, isDefault: value === null };

const applied = (patch: Patch): typeof settings => ({
  ...settings,
  pinWaitAfter: withValue(settings.pinWaitAfter, patch.pinWaitAfter),
  pinOffAfter: withValue(settings.pinOffAfter, patch.pinOffAfter),
});

const waitEnds = "2026-10-16T15:42:00Z";

const refusals: Record<string, () => Response> = {
  "1235": () => problem(401, "pin_wrong", "The email or PIN is wrong"),
  "0000": () =>
    problem(
      403,
      "pin_off",
      "This PIN is turned off. Use a passkey or ask an Admin for a Reset link.",
    ),
  "1111": () =>
    HttpResponse.json(
      {
        status: 429,
        code: "pin_wait",
        title: "Too many wrong PINs. Wait before trying again.",
        detail: waitEnds,
      },
      { status: 429, headers: { "Content-Type": "application/problem+json" } },
    ),
  "2222": () =>
    HttpResponse.json(
      {
        status: 429,
        code: "too_many_attempts",
        title: "Too many failed attempts from this address. Wait before trying again.",
        detail: waitEnds,
      },
      { status: 429, headers: { "Content-Type": "application/problem+json" } },
    ),
};

function reconfirmServer({ lapsesAgain = false } = {}) {
  let confirmed = false;
  const patches: Patch[] = [];
  return {
    handlers: [
      http.patch("*/api/v1/identity/settings", async ({ request }) => {
        const patch = UpdateIdentitySettingsBody.parse(await request.json());
        patches.push(patch);
        if (!confirmed || lapsesAgain) {
          return problem(403, "needs_reconfirm", "Confirm it's you to continue");
        }
        return HttpResponse.json(applied(patch));
      }),
      http.post("*/api/v1/session/reconfirm", async ({ request }) => {
        const { pin } = ReconfirmSessionBody.parse(await request.json());
        const refuse = refusals[pin];
        if (refuse) {
          return refuse();
        }
        confirmed = true;
        return new HttpResponse(null, { status: 204 });
      }),
    ],
    patches,
  };
}

const dialog = () => page.getByRole("dialog", { name: "Confirm it's you" });

async function typePin(pin: string) {
  await userEvent.click(dialog().getByRole("textbox", { name: "PIN", exact: true }));
  await userEvent.keyboard(pin);
}

async function opened(server: ReturnType<typeof reconfirmServer>) {
  renderApp({ path: "/library", handlers: server.handlers });
  await expect.element(page.getByRole("heading", { name: "Library" })).toBeVisible();
}

test("an action whose confirm lapsed asks for the PIN, then runs once more with the same body", async () => {
  const server = reconfirmServer();
  await opened(server);
  const saved = updateIdentitySettings({ pinWaitAfter: 3 });
  await expect.element(dialog()).toBeVisible();
  await expect.element(dialog().getByText("Enter your PIN to continue.")).toBeVisible();

  await typePin("1234");
  await expect(saved).resolves.toEqual(applied({ pinWaitAfter: 3 }));
  await expect.element(dialog()).not.toBeInTheDocument();
  expect(server.patches).toEqual([{ pinWaitAfter: 3 }, { pinWaitAfter: 3 }]);
});

test("Cancel fails the action with the Node's needs_reconfirm answer", async () => {
  const server = reconfirmServer();
  await opened(server);
  const refused = expect(updateIdentitySettings({ pinWaitAfter: 3 })).rejects.toMatchObject({
    name: "ApiError",
    status: 403,
    code: "needs_reconfirm",
    title: "Confirm it's you to continue",
  });
  await userEvent.click(dialog().getByRole("button", { name: "Cancel" }));

  await refused;
  await expect.element(dialog()).not.toBeInTheDocument();
  expect(server.patches).toHaveLength(1);
});

test("an action that still needs a confirm after the retry fails without asking again", async () => {
  const server = reconfirmServer({ lapsesAgain: true });
  await opened(server);
  const refused = expect(updateIdentitySettings({ pinWaitAfter: 3 })).rejects.toMatchObject({
    code: "needs_reconfirm",
  });
  await typePin("1234");

  await refused;
  await expect.element(dialog()).not.toBeInTheDocument();
  expect(server.patches).toHaveLength(2);
});

test("actions that ask at once share one dialog, and one PIN runs them all", async () => {
  const server = reconfirmServer();
  await opened(server);
  const first = updateIdentitySettings({ pinWaitAfter: 3 });
  const second = updateIdentitySettings({ pinOffAfter: 8 });
  await expect.element(dialog()).toBeVisible();
  await expect.poll(() => server.patches).toHaveLength(2);
  expect(page.getByRole("dialog").all()).toHaveLength(1);

  await typePin("1234");
  await expect(first).resolves.toMatchObject({ pinWaitAfter: { value: 3 } });
  await expect(second).resolves.toMatchObject({ pinOffAfter: { value: 8 } });
});

test.each([
  ["a wrong PIN", "1235", "That PIN is wrong."],
  [
    "a turned-off PIN",
    "0000",
    "This PIN is turned off. Use a passkey or ask an Admin for a Reset link.",
  ],
  [
    "a PIN in its wait",
    "1111",
    `Too many wrong PINs. Wait before trying again. You can try again at ${clock(waitEnds)}.`,
  ],
  [
    "an address past its limit",
    "2222",
    `Too many failed attempts from this address. Wait before trying again. You can try again at ${clock(waitEnds)}.`,
  ],
])("%s shows its alert in the dialog and clears the PIN", async (_case, pin, alert) => {
  const server = reconfirmServer();
  await opened(server);
  const refused = expect(updateIdentitySettings({ pinWaitAfter: 3 })).rejects.toMatchObject({
    code: "needs_reconfirm",
  });
  await typePin(pin);

  await expect.element(dialog().getByRole("alert")).toHaveTextContent(alert);
  await userEvent.click(dialog().getByText("Enter your PIN to continue."));
  await expect.element(dialog().getByRole("alert")).toHaveTextContent(alert);
  await expect.element(dialog().getByRole("textbox", { name: "PIN", exact: true })).toHaveValue("");
  await userEvent.click(dialog().getByRole("button", { name: "Cancel" }));
  await refused;
});

test("on a phone the dialog is a sheet with the keypad", async () => {
  await asPhone();
  const server = reconfirmServer();
  await opened(server);
  const saved = updateIdentitySettings({ pinWaitAfter: 3 });
  const keypad = dialog().getByRole("group", { name: "Keypad" });
  await expect.element(keypad).toBeVisible();
  for (const digit of "1234") {
    // oxlint-disable-next-line no-await-in-loop -- a person presses one key after another
    await userEvent.click(keypad.getByRole("button", { name: digit, exact: true }));
  }
  await expect(saved).resolves.toMatchObject({ pinWaitAfter: { value: 3 } });
});
