import { expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { SettingSpaceField } from "../api/gen/model";
import { renderKit } from "../test/render-kit";
import { SettingField } from "./setting-field";

const gb = 2 ** 30;
const tb = 2 ** 40;

const warnBelow: SettingSpaceField = {
  source: "database",
  applies: "live",
  pendingRestart: false,
  key: "health.storageWarnBelow",
  label: "Warn below",
  kind: "space",
  isDefault: true,
  defaultFrom: "Storage levels",
  value: { amount: 100 * gb, unit: "bytes" },
  running: { amount: 100 * gb, unit: "bytes" },
  default: { amount: 100 * gb, unit: "bytes" },
  maxBytes: 1024 * tb,
};

function saving() {
  return vi.fn<(value: unknown) => Promise<unknown>>(() => Promise.resolve());
}

test("a space shows its value in the largest whole unit, its default and its range", async () => {
  renderKit(
    <SettingField
      field={{ ...warnBelow, isDefault: false, value: { amount: 2 * tb, unit: "bytes" } }}
      onSave={saving()}
    />,
  );
  await expect.element(page.getByRole("textbox", { name: "Warn below" })).toHaveValue("2");
  await expect.element(page.getByRole("radio", { name: "TB" })).toBeChecked();
  await expect
    .element(page.getByText("Default 100 GB, from Storage levels. Up to 1024 TB or 100%."))
    .toBeVisible();
});

test("a space saves bytes in the unit chosen, or a percent", async () => {
  const onSave = saving();
  renderKit(<SettingField field={warnBelow} onSave={onSave} />);
  const input = page.getByRole("textbox", { name: "Warn below" });
  await userEvent.click(page.getByRole("radio", { name: "TB" }));
  await userEvent.fill(input, "1.5");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  expect(onSave).toHaveBeenLastCalledWith({ amount: 1536 * gb, unit: "bytes" });

  await userEvent.click(page.getByRole("radio", { name: "%" }));
  await userEvent.fill(input, "10");
  await userEvent.keyboard("{Enter}");
  expect(onSave).toHaveBeenLastCalledWith({ amount: 10, unit: "percent" });
});

test("a space refuses part of a percent and amounts out of range before any request", async () => {
  const onSave = saving();
  renderKit(<SettingField field={warnBelow} onSave={onSave} />);
  const input = page.getByRole("textbox", { name: "Warn below" });
  await userEvent.click(page.getByRole("radio", { name: "%" }));
  await userEvent.fill(input, "2.5");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByRole("alert")).toHaveTextContent("In whole percent.");

  await userEvent.fill(input, "101");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByRole("alert")).toHaveTextContent("Between 0 and 100%.");

  await userEvent.click(page.getByRole("radio", { name: "TB" }));
  await userEvent.fill(input, "1025");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByRole("alert")).toHaveTextContent("Between 0 GB and 1024 TB.");
  expect(onSave).not.toHaveBeenCalled();
});

test("a space off its default resets to null, and its row says the value", async () => {
  const onSave = saving();
  renderKit(
    <SettingField
      field={{ ...warnBelow, isDefault: false, value: { amount: 10, unit: "percent" } }}
      onSave={onSave}
      variant="row"
    />,
  );
  await userEvent.click(page.getByRole("button", { name: "Warn below: 10%" }));
  await userEvent.click(page.getByRole("button", { name: "Reset to default" }));
  expect(onSave).toHaveBeenCalledExactlyOnceWith(null);
});
