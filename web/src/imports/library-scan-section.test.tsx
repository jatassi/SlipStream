import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { SettingCountField, TasksTask } from "../api/gen/model";
import { UpdateLibraryScanSettingsBody } from "../api/gen/zod/imports/imports.zod";
import { renderApp } from "../test/render-app";
import { dailyCadence, housekeeping, taskServer } from "../test/tasks";

const share: SettingCountField = {
  key: "library.scan.stopShare",
  label: "Stop a scan when this share of a root's files is gone",
  kind: "count",
  source: "default",
  applies: "live",
  isDefault: true,
  pendingRestart: false,
  value: 25,
  running: 25,
  default: 25,
  min: 1,
  max: 100,
  unit: { one: "%", many: "%" },
};

const probes: SettingCountField = {
  ...share,
  key: "library.scan.probes",
  label: "Files each Node reads media info from at once",
  value: 2,
  running: 2,
  default: 2,
  max: 8,
  unit: { one: "file", many: "files" },
};

const libraryScan: TasksTask = {
  ...housekeeping,
  id: "library-scan",
  name: "Library scan",
  group: "store",
  cadence: {
    ...housekeeping.cadence,
    field: { ...dailyCadence, key: "tasks.library-scan.cadence", label: "Scan every" },
  },
};

function openSection(tasks: TasksTask[]) {
  const sent: unknown[] = [];
  let settings = [share, probes];
  renderApp({
    path: "/settings/library-scan",
    handlers: [
      ...taskServer({ tasks }).handlers,
      http.get("*/api/v1/imports/settings", () => HttpResponse.json({ settings })),
      http.patch("*/api/v1/imports/settings", async ({ request }) => {
        const { values } = UpdateLibraryScanSettingsBody.parse(await request.json());
        sent.push(values);
        settings = settings.map((field) => {
          const value = values[field.key];
          return typeof value === "number" ? { ...field, value, isDefault: false } : field;
        });
        return HttpResponse.json({ settings });
      }),
    ],
  });
  return { sent };
}

test("Library scan leads with what it does and its cadence, with the stop share and probes under Details", async () => {
  const { sent } = openSection([housekeeping, libraryScan]);
  await expect
    .element(
      page.getByText(
        "Library scan compares each library root with SlipStream's records. It adds the files it can match to Works, sends the rest to Pending imports and reads each file's media info. A scan that would mark too many of a root's files gone marks none and waits for you. The defaults suit most libraries.",
      ),
    )
    .toBeVisible();
  await expect.element(page.getByRole("textbox", { name: "Scan every" })).toBeVisible();
  await expect.element(page.getByRole("textbox", { name: probes.label })).not.toBeInTheDocument();
  await userEvent.click(page.getByRole("button", { name: "Details" }));
  await expect.element(page.getByRole("textbox", { name: share.label })).toHaveValue("25");
  const each = page.getByRole("textbox", { name: probes.label });
  await userEvent.clear(each);
  await userEvent.type(each, "4");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.poll(() => sent).toEqual([{ "library.scan.probes": 4 }]);
  await expect.element(each).toHaveValue("4");
  await expect
    .element(page.getByRole("link", { name: "Library roots" }).last())
    .toHaveAttribute("href", "/settings/library-roots");
});

test("before any root exists, Library scan asks for one in place of its cadence", async () => {
  openSection([housekeeping]);
  await expect.element(page.getByText("Add a library root to scan it.")).toBeVisible();
  await expect.element(page.getByRole("textbox", { name: "Scan every" })).not.toBeInTheDocument();
});
