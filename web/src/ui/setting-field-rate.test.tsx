import { expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { SettingRateField } from "../api/gen/model";
import { renderKit } from "../test/render-kit";
import { SettingField } from "./setting-field";

const about = { source: "database", applies: "live", pendingRestart: false } as const;

const downloadLimit: SettingRateField = {
  ...about,
  key: "downloading.downloadLimit",
  label: "Download limit",
  kind: "rate",
  isDefault: true,
  value: 0,
  running: 0,
  default: 0,
  max: 100 * 1024 * 1024,
  zero: "No limit",
};

function saving() {
  return vi.fn<(value: unknown) => Promise<unknown>>(() => Promise.resolve());
}

test("a rate of zero reads its words, and saves bytes a second in the unit chosen", async () => {
  const onSave = saving();
  renderKit(<SettingField field={downloadLimit} onSave={onSave} />);
  await expect
    .element(page.getByText("Default No limit. Up to 100 MB/s. 0 means no limit."))
    .toBeVisible();
  const input = page.getByRole("textbox", { name: "Download limit" });
  await expect.element(input).toHaveValue("0");
  await expect.element(page.getByText("No limit", { exact: true })).toBeVisible();

  await userEvent.fill(input, "1.5");
  await expect.element(page.getByRole("radio", { name: "MB/s" })).toBeChecked();
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  expect(onSave).toHaveBeenLastCalledWith(1_572_864);

  await userEvent.click(page.getByRole("radio", { name: "KB/s" }));
  await userEvent.fill(input, "512");
  await userEvent.keyboard("{Enter}");
  expect(onSave).toHaveBeenLastCalledWith(524_288);
});

test("a rate past its maximum or off a whole KB/s is refused before any request", async () => {
  const onSave = saving();
  renderKit(<SettingField field={downloadLimit} onSave={onSave} />);
  const input = page.getByRole("textbox", { name: "Download limit" });
  await userEvent.fill(input, "200");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByRole("alert")).toHaveTextContent("Up to 100 MB/s.");
  await expect.element(input).toHaveAttribute("aria-invalid", "true");

  await userEvent.click(page.getByRole("radio", { name: "KB/s" }));
  await userEvent.fill(input, "0.5");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByRole("alert")).toHaveTextContent("In whole KB/s.");
  expect(onSave).not.toHaveBeenCalled();
});

test("a rate row reads MB/s when whole and its zero words at zero", async () => {
  renderKit(
    <>
      <SettingField field={downloadLimit} onSave={saving()} variant="row" />
      <SettingField
        field={{ ...downloadLimit, label: "Upload limit", value: 2 * 1024 * 1024 }}
        onSave={saving()}
        variant="row"
      />
    </>,
  );
  await expect
    .element(page.getByRole("button", { name: "Download limit: No limit" }))
    .toBeVisible();
  await expect.element(page.getByRole("button", { name: "Upload limit: 2 MB/s" })).toBeVisible();
});
