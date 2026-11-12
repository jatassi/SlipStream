import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { SettingListField, SettingSizeField } from "../api/gen/model";
import { UpdateImportingSettingsBody } from "../api/gen/zod/imports/imports.zod";
import { renderApp } from "../test/render-app";

const extensions: SettingListField = {
  key: "library.importing.videoExtensions",
  label: "Video file extensions",
  kind: "list",
  source: "default",
  applies: "live",
  isDefault: true,
  pendingRestart: false,
  value: ["mkv", "avi"],
  running: ["mkv", "avi"],
  default: ["mkv", "avi"],
  min: 1,
  item: "extension",
};

const smallest: SettingSizeField = {
  key: "library.importing.smallestVideo",
  label: "Smallest video file",
  kind: "size",
  source: "default",
  applies: "live",
  isDefault: true,
  pendingRestart: false,
  value: 52_428_800,
  running: 52_428_800,
  default: 52_428_800,
  min: 0,
  max: 10_737_418_240,
};

function openSection() {
  const sent: unknown[] = [];
  const asked: string[][] = [];
  renderApp({
    path: "/settings/importing",
    handlers: [
      http.get(
        "*/api/v1/imports/importing-settings/extension-impact",
        ({ request }) => {
          const listed = new URL(request.url).searchParams
            .getAll("extensions")
            .flatMap((part) => part.split(","));
          asked.push(listed);
          return HttpResponse.json(
            listed.includes("avi")
              ? { files: 0, pending: 0 }
              : { files: 3, pending: 1 },
          );
        },
      ),
      http.get("*/api/v1/imports/importing-settings", () =>
        HttpResponse.json({ settings: [extensions, smallest] }),
      ),
      http.patch("*/api/v1/imports/importing-settings", async ({ request }) => {
        const { values } = UpdateImportingSettingsBody.parse(
          await request.json(),
        );
        sent.push(values);
        return HttpResponse.json({ settings: [extensions, smallest] });
      }),
    ],
  });
  return { sent, asked };
}

test("dropping an extension names the files it stops counting before it saves", async () => {
  const { sent, asked } = openSection();
  await userEvent.click(page.getByRole("button", { name: "Details" }));
  await userEvent.click(page.getByRole("button", { name: "Remove avi" }));
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect
    .element(page.getByRole("status"))
    .toHaveTextContent(
      "This list stops counting 3 library files and 1 file in Pending imports as video.",
    );
  expect(asked).toEqual([["mkv"]]);
  expect(sent).toEqual([]);
  await userEvent.click(page.getByRole("button", { name: "Save anyway" }));
  await expect
    .poll(() => sent)
    .toEqual([{ "library.importing.videoExtensions": ["mkv"] }]);
});

test("adding an extension saves at once, and Library scan is a link away", async () => {
  const { sent } = openSection();
  await userEvent.click(page.getByRole("button", { name: "Details" }));
  await userEvent.type(
    page.getByRole("textbox", { name: "Video file extensions" }),
    "webm",
  );
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect
    .poll(() => sent)
    .toEqual([{ "library.importing.videoExtensions": ["mkv", "avi", "webm"] }]);
  await expect
    .element(page.getByRole("link", { name: "Library scan" }).last())
    .toHaveAttribute("href", "/settings/library-scan");
});
