import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { SettingDurationField, SettingSwitchField, WorksContentType } from "../api/gen/model";
import { getGetContentTypeRefreshMockHandler } from "../api/gen/query/metadata/metadata.msw";
import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { cinema } from "../test/library";
import { profileHandlers } from "../test/profiles";
import { renderApp } from "../test/render-app";
import { archiveRoot, filmsRoot, refused } from "../test/roots";

const about = { source: "database", applies: "live", pendingRestart: false } as const;

const monitorOnAdd: SettingSwitchField = {
  ...about,
  kind: "switch",
  key: "cinema.add.monitor",
  label: "Monitor new Films",
  isDefault: true,
  value: false,
  running: false,
  default: false,
};

const cinemaDelay: SettingDurationField = {
  ...about,
  kind: "duration",
  key: "cinema.delay",
  label: "Cinema delay",
  isDefault: true,
  value: 90 * 86_400,
  running: 90 * 86_400,
  default: 90 * 86_400,
  min: 0,
  max: 365 * 86_400,
  step: 86_400,
};

const zines: WorksContentType = {
  ...cinema,
  id: "zine",
  name: "Zines",
  enabled: false,
  work: { one: "Zine", many: "Zines" },
};

const handlers = (more: Parameters<typeof renderApp>[0] = {}) => [
  ...(more.handlers ?? []),
  getListContentTypesMockHandler({ contentTypes: [cinema, zines] }),
  ...profileHandlers(),
  http.get("*/api/v1/roots", () => HttpResponse.json({ roots: [filmsRoot, archiveRoot] })),
  getGetContentTypeRefreshMockHandler({ lastRunAt: null, running: false }),
  http.get("*/api/v1/content-types/cinema/settings", () =>
    HttpResponse.json({ settings: [cinemaDelay, monitorOnAdd] }),
  ),
];

test("each Content Type is a row with its switch, and an on type shows its settings with Monitor first", async () => {
  renderApp({ path: "/settings/content-types", handlers: handlers() });
  const films = page.getByRole("article", { name: "Films" });
  await expect.element(films.getByRole("switch", { name: "Films" })).toBeChecked();
  await expect
    .element(films.getByRole("combobox", { name: "Default Profile" }))
    .toHaveTextContent("Films");
  await expect
    .element(films.getByRole("combobox", { name: "Default root" }))
    .toHaveTextContent("/srv/media/films");
  await expect.element(films.getByText("It hasn't run yet.", { exact: false })).toBeVisible();
  const labels = films
    .getByRole("switch")
    .all()
    .map((control) => control.element().closest("label")?.textContent);
  expect(labels).toEqual(["Films", "Monitor new Films"]);

  const off = page.getByRole("article", { name: "Zines" });
  await expect.element(off.getByRole("switch", { name: "Zines" })).not.toBeChecked();
  await expect.element(off.getByRole("button", { name: "Refresh all" })).not.toBeInTheDocument();
});

test("a type's setting saves under its key", async () => {
  const sent: unknown[] = [];
  renderApp({
    path: "/settings/content-types",
    handlers: handlers({
      handlers: [
        http.patch("*/api/v1/content-types/cinema/settings", async ({ request }) => {
          sent.push(await request.json());
          return HttpResponse.json({
            settings: [cinemaDelay, { ...monitorOnAdd, value: true, isDefault: false }],
          });
        }),
      ],
    }),
  });
  await userEvent.click(page.getByRole("switch", { name: "Monitor new Films" }));
  await expect.poll(() => sent).toEqual([{ values: { "cinema.add.monitor": true } }]);
});

test("turning off a type that holds Works is refused with a link to its Library", async () => {
  const sent: unknown[] = [];
  renderApp({
    path: "/settings/content-types",
    handlers: handlers({
      handlers: [
        http.put("*/api/v1/content-types/cinema/enabled", async ({ request }) => {
          sent.push(await request.json());
          return refused(
            409,
            "content_type_in_use",
            "Remove its Works before turning this Content Type off",
          );
        }),
      ],
    }),
  });
  const films = page.getByRole("switch", { name: "Films" });
  await userEvent.click(films);
  await expect.poll(() => sent).toEqual([{ enabled: false }]);
  const alert = page.getByRole("alert");
  await expect
    .element(alert)
    .toHaveTextContent(
      "The Library still holds Films. Delete them in the Films Library, then turn Films off.",
    );
  await expect
    .element(alert.getByRole("link", { name: "Films Library" }))
    .toHaveAttribute("href", "/library/cinema");
  await expect.element(films).toBeChecked();
});

test("Make default in a type's row makes that root the default for new Works", async () => {
  const made: string[] = [];
  renderApp({
    path: "/settings/content-types",
    handlers: handlers({
      handlers: [
        http.post("*/api/v1/roots/:rootId/default", ({ params }) => {
          made.push(String(params["rootId"]));
          return new HttpResponse(null, { status: 204 });
        }),
      ],
    }),
  });
  await userEvent.click(page.getByRole("combobox", { name: "Default root" }));
  await userEvent.click(page.getByRole("option", { name: "/mnt/archive/films" }));
  await expect.element(page.getByText("New Films now go into /mnt/archive/films.")).toBeVisible();
  expect(made).toEqual([archiveRoot.id]);
});

test("Refresh all starts a run and reads its last run", async () => {
  let started = 0;
  renderApp({
    path: "/settings/content-types",
    handlers: handlers({
      handlers: [
        getGetContentTypeRefreshMockHandler(() =>
          started === 0
            ? { lastRunAt: "2026-05-03T14:20:00Z", running: false }
            : { lastRunAt: "2026-05-03T14:20:00Z", running: true },
        ),
        http.post("*/api/v1/content-types/cinema/refresh", () => {
          started += 1;
          return HttpResponse.json({ runId: 7 }, { status: 202 });
        }),
      ],
    }),
  });
  await expect.element(page.getByText("It last ran", { exact: false })).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Refresh all" }));
  await expect.element(page.getByText("Refreshing every Film from TMDB.")).toBeVisible();
  await expect.element(page.getByText("It's running now.", { exact: false })).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Refresh all" })).toBeDisabled();
  expect(started).toBe(1);
});
