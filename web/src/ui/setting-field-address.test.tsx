import { expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { SettingAddressField } from "../api/gen/model";
import { renderKit } from "../test/render-kit";
import { SettingField } from "./setting-field";

const about = { source: "database", applies: "live", pendingRestart: false } as const;

const listen: SettingAddressField = {
  ...about,
  key: "torrent.listen",
  label: "Listen address",
  kind: "address",
  applies: "restart",
  isDefault: true,
  value: "0.0.0.0:6881",
  running: "0.0.0.0:6881",
  default: "0.0.0.0:6881",
};

function saving() {
  return vi.fn<(value: unknown) => Promise<unknown>>(() => Promise.resolve());
}

test("an address edits its address and port apart and saves them as one", async () => {
  const onSave = saving();
  renderKit(<SettingField field={listen} onSave={onSave} />);
  await expect.element(page.getByText("Default 0.0.0.0:6881.")).toBeVisible();
  await expect.element(page.getByText("Applies after restart.")).toBeVisible();
  const address = page.getByRole("textbox", { name: "Address" });
  const port = page.getByRole("textbox", { name: "Port" });
  await expect.element(address).toHaveValue("0.0.0.0");
  await expect.element(port).toHaveValue("6881");

  await userEvent.fill(port, "0");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByRole("alert")).toHaveTextContent("A port between 1 and 65535.");
  expect(onSave).not.toHaveBeenCalled();

  await userEvent.fill(port, "51413");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  expect(onSave).toHaveBeenLastCalledWith("0.0.0.0:51413");

  await userEvent.fill(address, "::");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  expect(onSave).toHaveBeenLastCalledWith("[::]:51413");
});

test("an IPv6 address shows without its brackets", async () => {
  renderKit(<SettingField field={{ ...listen, value: "[fd00::2]:6881" }} onSave={saving()} />);
  await expect.element(page.getByRole("textbox", { name: "Address" })).toHaveValue("fd00::2");
  await expect.element(page.getByRole("textbox", { name: "Port" })).toHaveValue("6881");
  expect(page.getByRole("button", { name: "Save" }).query()).toBeNull();
});
