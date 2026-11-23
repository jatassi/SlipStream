import { HttpResponse } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderApp } from "../test/render-app";
import { refused } from "../test/roots";
import { subdl } from "../test/subtitles";
import { subtitlesWorld } from "../test/subtitles-world";

const path = "/settings/subtitles";

const rejected =
  'SubDL rejected this key. It responded "Not Authorized". Check that you copied the whole key.';

const world = () =>
  subtitlesWorld({
    key: (_id, key) =>
      key === "good-key"
        ? HttpResponse.json({
            ...subdl,
            hasKey: true,
            position: 2,
            account: "jsmith",
            plan: "Free",
            allowance: {
              at: new Date().toISOString(),
              searches: null,
              downloads: {
                used: 31,
                limit: 50,
                remaining: 19,
                resets: new Date(Date.now() + 3_600_000).toISOString(),
                exact: true,
              },
            },
            state: { kind: "ready", since: null, until: null, words: null },
          })
        : refused(
            422,
            "subtitle_key_rejected",
            "The Subtitle Provider rejected the key, so SlipStream didn't save it",
            {
              detail: rejected,
              errors: [{ location: "body.key", message: rejected }],
            },
          ),
  });

async function openDialog() {
  await userEvent.click(
    page
      .getByRole("list", { name: "Subtitle Providers" })
      .getByRole("button", { name: "Add key" })
      .first(),
  );
  return page.getByRole("dialog", { name: "Add your SubDL key" });
}

test("the three steps link SubDL's key page and say the password is never asked for", async () => {
  renderApp({ path, handlers: world().handlers });
  const dialog = await openDialog();
  const steps = dialog.getByRole("list", { name: "Steps to add your SubDL key" });
  await expect
    .element(
      steps.getByText("Open SubDL's API page. Sign in, or create a free account.", {
        exact: false,
      }),
    )
    .toBeVisible();
  await expect
    .element(steps.getByText("Generate a key and copy it.", { exact: false }))
    .toBeVisible();
  await expect.element(steps.getByText("Paste the key here.", { exact: false })).toBeVisible();
  await expect
    .element(steps.getByRole("link", { name: "Open SubDL" }))
    .toHaveAttribute("href", subdl.keyPage);
  await expect
    .element(dialog.getByText("SlipStream never asks for your SubDL password."))
    .toBeVisible();
});

test("an accepted key replaces the steps with the account, its plan and its allowance, and the place in the order", async () => {
  const subtitles = world();
  renderApp({ path, handlers: subtitles.handlers });
  const dialog = await openDialog();
  await userEvent.fill(dialog.getByRole("textbox", { name: "Key" }), "good-key");
  await userEvent.click(dialog.getByRole("button", { name: "Check key" }));
  await expect
    .element(
      dialog.getByText(
        "The key works. It belongs to jsmith on SubDL's Free plan, with 19 of 50 downloads left. SubDL is second in the search order.",
      ),
    )
    .toBeVisible();
  await expect.element(dialog.getByRole("list", { name: /Steps/u })).not.toBeInTheDocument();
  expect(subtitles.writes.at(-1)).toEqual({
    path: "/api/v1/subtitles/providers/subdl/key",
    body: { key: "good-key" },
  });
});
