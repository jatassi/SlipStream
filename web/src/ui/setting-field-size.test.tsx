import { expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { SettingCountField, SettingSizeField } from "../api/gen/model";
import { renderKit } from "../test/render-kit";
import { SettingField } from "./setting-field";

const mb = 2 ** 20;
const gb = 2 ** 30;

const about = { source: "database", applies: "live", pendingRestart: false } as const;

const smallestVideo: SettingSizeField = {
  ...about,
  key: "library.importing.smallestVideo",
  label: "Smallest video file",
  kind: "size",
  isDefault: true,
  value: 50 * mb,
  running: 50 * mb,
  default: 50 * mb,
  min: 0,
  max: 10 * gb,
};

const sampleShorter: SettingCountField = {
  ...about,
  key: "library.importing.sampleShorter",
  label: "Sample when shorter than",
  kind: "count",
  isDefault: true,
  value: 25,
  running: 25,
  default: 25,
  min: 0,
  max: 90,
  unit: { one: "% of its runtime", many: "% of its runtime" },
  zero: "Off",
};

function saving() {
  return vi.fn<(value: unknown) => Promise<unknown>>(() => Promise.resolve());
}

test("a size shows its default and range, and saves bytes in the unit chosen", async () => {
  const onSave = saving();
  renderKit(<SettingField field={smallestVideo} onSave={onSave} />);
  await expect.element(page.getByText("Default 50 MB. Between 0 MB and 10 GB.")).toBeVisible();
  const input = page.getByRole("textbox", { name: "Smallest video file" });
  await expect.element(input).toHaveValue("50");
  await expect.element(page.getByRole("radio", { name: "MB" })).toBeChecked();

  await userEvent.click(page.getByRole("radio", { name: "GB" }));
  await userEvent.fill(input, "1.5");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  expect(onSave).toHaveBeenLastCalledWith(1536 * mb);
});

test("a size past its maximum or off a whole MB is refused before any request", async () => {
  const onSave = saving();
  renderKit(<SettingField field={smallestVideo} onSave={onSave} />);
  const input = page.getByRole("textbox", { name: "Smallest video file" });
  await userEvent.click(page.getByRole("radio", { name: "GB" }));
  await userEvent.fill(input, "11");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByRole("alert")).toHaveTextContent("Between 0 MB and 10 GB.");
  await expect.element(input).toHaveAttribute("aria-invalid", "true");

  await userEvent.click(page.getByRole("radio", { name: "MB" }));
  await userEvent.fill(input, "0.5");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByRole("alert")).toHaveTextContent("In whole MB.");
  expect(onSave).not.toHaveBeenCalled();
});

test("a size row reads GB when whole", async () => {
  renderKit(
    <SettingField field={{ ...smallestVideo, value: 2 * gb }} onSave={saving()} variant="row" />,
  );
  await expect
    .element(page.getByRole("button", { name: "Smallest video file: 2 GB" }))
    .toBeVisible();
});

test("a share of a runtime reads as one unit after the number, and 0 reads off", async () => {
  renderKit(<SettingField field={sampleShorter} onSave={saving()} />);
  await expect
    .element(
      page.getByText("Default 25% of its runtime. Between 0 and 90% of its runtime. 0 means off."),
    )
    .toBeVisible();
  await expect.element(page.getByText("% of its runtime", { exact: true })).toBeVisible();
  await userEvent.fill(page.getByRole("textbox", { name: "Sample when shorter than" }), "0");
  await expect.element(page.getByText("Off", { exact: true })).toBeVisible();
});
