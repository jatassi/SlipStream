import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page } from "vitest/browser";

import type {
  SettingChoiceField,
  SettingDurationField,
  SettingSwitchField,
} from "../api/gen/model";
import { getGetContentTypeRefreshMockHandler } from "../api/gen/query/metadata/metadata.msw";
import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { anthologies } from "../test/anthology";
import { films, profileHandlers } from "../test/profiles";
import { renderApp } from "../test/render-app";
import { filmsRoot } from "../test/roots";

const about = { source: "database", applies: "live", pendingRestart: false } as const;

const toggle = (key: string, label: string, value: boolean): SettingSwitchField => ({
  ...about,
  kind: "switch",
  key,
  label,
  isDefault: true,
  value,
  running: value,
  default: value,
});

const monitorOnAdd: SettingChoiceField = {
  ...about,
  kind: "choice",
  key: "anthology.add.monitor",
  label: "Stories to monitor",
  isDefault: true,
  value: "all",
  running: "all",
  default: "all",
  options: [
    { value: "all", label: "All" },
    { value: "future", label: "Future" },
    { value: "none", label: "None" },
  ],
};

const publicationDelay: SettingDurationField = {
  ...about,
  kind: "duration",
  key: "anthology.publicationDelay",
  label: "Delay after a publication date",
  isDefault: true,
  value: 86_400,
  running: 86_400,
  default: 86_400,
  min: 0,
  max: 30 * 86_400,
  step: 86_400,
};

const handlers = [
  getListContentTypesMockHandler({ contentTypes: [anthologies] }),
  ...profileHandlers([{ ...films, contentType: anthologies.id, default: true }]),
  http.get("*/api/v1/roots", () =>
    HttpResponse.json({ roots: [{ ...filmsRoot, contentType: anthologies.id }] }),
  ),
  getGetContentTypeRefreshMockHandler({ lastRunAt: null, running: false }),
  http.get("*/api/v1/content-types/anthology/settings", () =>
    HttpResponse.json({
      settings: [
        publicationDelay,
        toggle("anthology.add.levelFolders", "Volume folders", true),
        monitorOnAdd,
        toggle("anthology.add.asides", "Include extras", false),
      ],
    }),
  ),
];

test("a many-Items type's row orders Profile, root, the add defaults, then its own settings and Refresh all", async () => {
  renderApp({ path: "/settings/content-types", handlers });
  const row = page.getByRole("article", { name: "Anthologies" });
  const inOrder = [
    row.getByRole("combobox", { name: "Default Profile" }),
    row.getByRole("combobox", { name: "Default root" }),
    row.getByRole("radiogroup", { name: "Stories to monitor" }),
    row.getByRole("switch", { name: "Include extras" }),
    row.getByRole("switch", { name: "Volume folders" }),
    row.getByLabelText("Delay after a publication date"),
    row.getByRole("button", { name: "Refresh all" }),
  ];
  await expect.element(row.getByRole("button", { name: "Refresh all" })).toBeVisible();
  const shown = inOrder.map((control) => control.element());
  const sorted = shown.toSorted((a, b) =>
    a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING ? -1 : 1,
  );
  expect(sorted).toEqual(shown);
});
