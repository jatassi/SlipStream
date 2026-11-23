import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { SubtitlesProvider } from "../api/gen/model";
import { renderApp } from "../test/render-app";
import { providerLine, providersCaption } from "../subtitles/settings-words";
import { gestdown, movies, subdl, subsource, tv } from "../test/subtitles";
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

test("each row reads its provider's state, with its downloads meter and its switch", async () => {
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
      on: false,
      state: { kind: "off", since: null, until: null, words: null },
    },
    { ...gestdown, state: { kind: "failing", since: anHourAgo, until: null, words: "timeout" } },
  ];
  renderApp({ path, handlers: subtitlesWorld({ providers }).handlers });
  await expect.element(list()).toHaveAccessibleDescription(providersCaption);
  const types = [movies, tv];
  const clock = { now: new Date(), zone: undefined };
  await Promise.all(
    providers.map((provider, index) =>
      expect
        .element(list().getByRole("listitem").nth(index))
        .toMatchTextContent(providerLine(provider, types, clock)),
    ),
  );
  await expect
    .element(list().getByRole("meter", { name: "SubDL downloads left" }))
    .toHaveAttribute("aria-valuenow", "19");
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
