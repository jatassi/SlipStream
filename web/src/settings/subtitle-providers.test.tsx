import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { SubtitlesProvider } from "../api/gen/model";
import { renderApp } from "../test/render-app";
import { asPhone } from "../test/viewport";
import { gestdown, subdl, subsource } from "../test/subtitles";
import { subtitlesWorld } from "../test/subtitles-world";

const path = "/settings/subtitles";

const list = () => page.getByRole("list", { name: "Subtitle Providers" });

const names = () =>
  list()
    .getByRole("listitem")
    .all()
    .map((row) => row.element().querySelector(".font-semibold")?.textContent);

const inAnHour = new Date(Date.now() + 3_600_000).toISOString();
const anHourAgo = new Date(Date.now() - 3_600_000).toISOString();

test("the caption says what the order does, and Gestdown reads TV only with no key", async () => {
  renderApp({ path, handlers: subtitlesWorld().handlers });
  await expect
    .element(list())
    .toHaveAccessibleDescription(
      "SlipStream searches these providers in order. When two subtitles match the file equally well, SlipStream picks the one from the provider higher on the list.",
    );
  await expect.element(list().getByText("TV only. No key needed.", { exact: true })).toBeVisible();
  await expect
    .element(list().getByText("SubDL needs your free key. Add it to search Movies and TV."))
    .toBeVisible();
});

test("each state reads as its own sentence", async () => {
  const providers: SubtitlesProvider[] = [
    {
      ...subdl,
      hasKey: true,
      allowance: {
        at: anHourAgo,
        searches: null,
        downloads: { used: 31, limit: 50, remaining: 19, resets: inAnHour, exact: true },
      },
      state: { kind: "ready", since: null, until: null, words: null },
    },
    {
      ...subsource,
      hasKey: true,
      state: { kind: "rejected", since: anHourAgo, until: null, words: "Invalid API key" },
    },
    { ...gestdown, state: { kind: "failing", since: anHourAgo, until: null, words: "timeout" } },
  ];
  renderApp({ path, handlers: subtitlesWorld({ providers }).handlers });
  await expect
    .element(list().getByText(/^Ready, with 19 of 50 downloads left\. The count resets .+\.$/u))
    .toBeVisible();
  await expect
    .element(list().getByRole("meter", { name: "SubDL downloads left" }))
    .toHaveAttribute("aria-valuenow", "19");
  await expect
    .element(
      list().getByText(
        /^SubSource rejected the key (?:today|yesterday) at .+ and responded "Invalid API key"\. Replace the key\.$/u,
      ),
    )
    .toBeVisible();
  await expect
    .element(
      list().getByText(
        /^Gestdown hasn't responded since .+\. SlipStream retries automatically\. TV only\. No key needed\.$/u,
      ),
    )
    .toBeVisible();
});

test("out of downloads and off read as their own sentences", async () => {
  const providers: SubtitlesProvider[] = [
    { ...subdl, hasKey: true, state: { kind: "out", since: null, until: inAnHour, words: null } },
    { ...subsource, on: false, state: { kind: "off", since: null, until: null, words: null } },
    gestdown,
  ];
  renderApp({ path, handlers: subtitlesWorld({ providers }).handlers });
  await expect
    .element(list().getByText(/^SubDL is out of downloads\. SlipStream searches it again .+\.$/u))
    .toBeVisible();
  await expect.element(list().getByText("Off. SlipStream doesn't search SubSource.")).toBeVisible();
  await expect.element(list().getByRole("switch", { name: "Search SubSource" })).not.toBeChecked();
});

test("the move buttons and the arrow keys reorder the providers and send the whole order", async () => {
  const world = subtitlesWorld();
  renderApp({ path, handlers: world.handlers });
  await expect.poll(names).toEqual(["SubDL", "SubSource", "Gestdown"]);
  await userEvent.click(list().getByRole("button", { name: "Move Gestdown up" }));
  await expect.poll(names).toEqual(["SubDL", "Gestdown", "SubSource"]);
  await expect
    .poll(() => world.writes.at(-1)?.body)
    .toEqual({ ids: ["subdl", "gestdown", "subsource"] });

  const handle = list().getByRole("button", { name: "Move SubDL", exact: true });
  handle.element().focus();
  await userEvent.keyboard("{ArrowDown}");
  await expect.poll(names).toEqual(["Gestdown", "SubDL", "SubSource"]);
  await expect
    .poll(() => world.writes.at(-1)?.body)
    .toEqual({ ids: ["gestdown", "subdl", "subsource"] });
});

test("the switch turns a provider off", async () => {
  const world = subtitlesWorld();
  renderApp({ path, handlers: world.handlers });
  await userEvent.click(list().getByRole("switch", { name: "Search Gestdown" }));
  await expect
    .poll(() => world.writes.at(-1))
    .toEqual({
      path: "/api/v1/subtitles/providers/gestdown",
      body: { on: false },
    });
});

test("on a phone the moves sit in one 44px row action", async () => {
  await asPhone();
  renderApp({ path, handlers: subtitlesWorld().handlers });
  const actions = list().getByRole("button", { name: "Actions for SubSource" });
  await expect.element(actions).toBeVisible();
  expect(actions.element().getBoundingClientRect().height).toBeGreaterThanOrEqual(44);
  await userEvent.click(actions);
  await expect.element(page.getByRole("button", { name: "Move up" })).toBeVisible();
});
