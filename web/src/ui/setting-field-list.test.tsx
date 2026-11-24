import { expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { SettingListField } from "../api/gen/model";
import { renderKit } from "../test/render-kit";
import { SettingField } from "./setting-field";

const videoExtensions: SettingListField = {
  source: "database",
  applies: "live",
  pendingRestart: false,
  key: "library.importing.videoExtensions",
  label: "Video file extensions",
  kind: "list",
  isDefault: true,
  value: ["mkv", "mp4"],
  running: ["mkv", "mp4"],
  default: ["mkv", "mp4"],
  min: 1,
  item: "extension",
};

function saving() {
  return vi.fn<(value: unknown) => Promise<unknown>>(() => Promise.resolve());
}

test("a list shows each entry as a chip, its default and its fewest entries", async () => {
  renderKit(<SettingField field={videoExtensions} onSave={saving()} />);
  const chips = page.getByRole("list", { name: "Video file extensions" }).getByRole("listitem");
  await expect.element(chips.nth(0)).toHaveTextContent("mkv");
  await expect.element(chips.nth(1)).toHaveTextContent("mp4");
  await expect.element(page.getByText("Default mkv, mp4. At least 1 extension.")).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Save" })).not.toBeInTheDocument();
});

test("typed extensions are written lowercase with no dot and saved in order", async () => {
  const onSave = saving();
  renderKit(<SettingField field={videoExtensions} onSave={onSave} />);
  const input = page.getByRole("textbox", { name: "Video file extensions" });
  await userEvent.type(input, ".WEBM,");
  await expect.element(page.getByRole("button", { name: "Remove webm" })).toBeVisible();
  await expect.element(input).toHaveValue("");

  await userEvent.click(page.getByRole("button", { name: "Remove mkv" }));
  await userEvent.type(input, "Avi");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  expect(onSave).toHaveBeenLastCalledWith(["mp4", "webm", "avi"]);
});

test("a duplicate, a pattern or an empty list is refused before any request", async () => {
  const onSave = saving();
  renderKit(<SettingField field={videoExtensions} onSave={onSave} />);
  const input = page.getByRole("textbox", { name: "Video file extensions" });
  await userEvent.type(input, "MKV{Enter}");
  await expect.element(page.getByRole("alert")).toMatchTextContent(/^mkv is already/u);
  await expect.element(input).toHaveValue("MKV");

  await userEvent.fill(input, "*.ts");
  await userEvent.keyboard("{Enter}");
  await expect
    .element(page.getByRole("alert"))
    .toHaveTextContent(`"*.ts" isn't an extension. Use letters and digits only, such as mkv.`);

  await userEvent.fill(input, "");
  await userEvent.keyboard("{Backspace}{Backspace}");
  await expect.element(page.getByRole("list")).not.toBeInTheDocument();
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByRole("alert")).toMatchTextContent(/at least 1 extension/u);
  expect(onSave).not.toHaveBeenCalled();
});

test("a list row reads its entries", async () => {
  renderKit(<SettingField field={videoExtensions} onSave={saving()} variant="row" />);
  await expect
    .element(page.getByRole("button", { name: "Video file extensions: mkv, mp4" }))
    .toBeVisible();
});

const stunServers: SettingListField = {
  ...videoExtensions,
  key: "reachability.stunServers",
  label: "STUN servers",
  isDefault: false,
  value: [],
  running: [],
  default: [],
  min: 0,
  item: "host-port",
};

test.each([
  ["STUN.Example.org:3478", "stun.example.org:3478"],
  ["192.0.2.7:19302", "192.0.2.7:19302"],
  ["[2001:DB8::1]:3478", "[2001:db8::1]:3478"],
  ["stun.example.org", undefined],
  ["2001:db8::1", undefined],
  ["stun.example.org:0", undefined],
  ["stun.example.org:70000", undefined],
  ["stun://stun.example.org:3478", undefined],
])("a server typed as %s is saved as %s", async (typed, saved) => {
  const onSave = saving();
  renderKit(<SettingField field={stunServers} onSave={onSave} />);
  await userEvent.fill(page.getByRole("textbox", { name: "STUN servers" }), typed);
  await userEvent.keyboard("{Enter}");
  if (saved === undefined) {
    await expect.element(page.getByRole("alert")).toBeVisible();
    expect(onSave).not.toHaveBeenCalled();
    return;
  }
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  expect(onSave).toHaveBeenLastCalledWith([saved]);
});
