import { expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import type {
  SettingChoiceField,
  SettingCountField,
  SettingListenField,
  SettingSwitchField,
} from "../api/gen/model";
import { renderKit } from "../test/render-kit";
import { SettingField } from "./setting-field";

const about = { source: "config.yaml", applies: "live", pendingRestart: false } as const;

const allNetworks = { host: "all", port: 8484, address: "[::]:8484" } as const;

const listen: SettingListenField = {
  ...about,
  key: "http.listen",
  label: "Listen on",
  kind: "listen",
  applies: "restart",
  isDefault: true,
  value: allNetworks,
  running: allNetworks,
  default: allNetworks,
};

const daysKept: SettingCountField = {
  ...about,
  key: "log.max_age_days",
  label: "Days kept",
  kind: "count",
  isDefault: false,
  value: 0,
  running: 0,
  default: 30,
  min: 0,
  max: 3650,
  unit: { one: "day", many: "days" },
  zero: "No limit",
};

const compress: SettingSwitchField = {
  ...about,
  key: "log.compress",
  label: "Compress rotated files",
  kind: "switch",
  isDefault: true,
  value: true,
  running: true,
  default: true,
};

const level: SettingChoiceField = {
  ...about,
  key: "log.level",
  label: "Level",
  kind: "choice",
  isDefault: true,
  value: "info",
  running: "info",
  default: "info",
  options: [
    { value: "debug", label: "Debug" },
    { value: "info", label: "Info" },
    { value: "warn", label: "Warn" },
    { value: "error", label: "Error" },
  ],
};

const saving = () => vi.fn<(value: unknown) => Promise<unknown>>(() => Promise.resolve());

test("each Node kind is a control named for its setting", async () => {
  renderKit(
    <>
      <SettingField field={listen} onSave={saving()} />
      <SettingField field={daysKept} onSave={saving()} />
      <SettingField field={compress} onSave={saving()} />
      <SettingField field={level} onSave={saving()} />
    </>,
  );
  await expect.element(page.getByRole("textbox", { name: "Port" })).toHaveValue("8484");
  await expect.element(page.getByRole("textbox", { name: "Days kept" })).toHaveValue("0");
  await expect.element(page.getByRole("switch", { name: "Compress rotated files" })).toBeChecked();
  await expect.element(page.getByRole("combobox", { name: "Level" })).toHaveTextContent("Info");
});

test("a listen address shows its default, refuses a port out of range and saves an address", async () => {
  const onSave = saving();
  renderKit(<SettingField field={listen} onSave={onSave} />);
  await expect.element(page.getByText("Default All networks, port 8484.")).toBeVisible();
  await expect.element(page.getByText("Applies after restart.")).toBeVisible();
  const input = page.getByRole("textbox", { name: "Port" });
  await userEvent.fill(input, "70000");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByRole("alert")).toHaveTextContent("Between 1 and 65535.");
  expect(onSave).not.toHaveBeenCalled();

  await userEvent.fill(input, "9000");
  await userEvent.click(page.getByRole("radio", { name: "This machine only" }));
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  expect(onSave).toHaveBeenCalledExactlyOnceWith("[::1]:9000");
});

test("the All networks warning shows only once All networks is chosen in place of this machine only", async () => {
  const local = { host: "local", port: 8484, address: "[::1]:8484" } as const;
  const onSave = saving();
  renderKit(
    <SettingField
      field={{ ...listen, isDefault: false, value: local, running: local }}
      onSave={onSave}
    />,
  );
  const warning = page.getByText(/Over IPv6/u);
  await expect.element(page.getByRole("radio", { name: "This machine only" })).toBeChecked();
  expect(warning.query()).toBeNull();

  await userEvent.click(page.getByRole("radio", { name: "All networks" }));
  await expect.element(warning).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  expect(onSave).toHaveBeenCalledExactlyOnceWith("[::]:8484");
});

test("an address set by hand in config.yaml keeps its host and saves a new port", async () => {
  const handSet = { host: "other", port: 8484, address: "192.168.1.20:8484" } as const;
  const onSave = saving();
  renderKit(
    <SettingField field={{ ...listen, value: handSet, running: handSet }} onSave={onSave} />,
  );
  await expect.element(page.getByText("192.168.1.20", { exact: true })).toBeVisible();
  expect(page.getByRole("radio").query()).toBeNull();
  await userEvent.fill(page.getByRole("textbox", { name: "Port" }), "9000");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  expect(onSave).toHaveBeenCalledExactlyOnceWith("192.168.1.20:9000");
});

test("a key set by the environment is locked and names its variable", async () => {
  renderKit(
    <>
      <SettingField
        field={{ ...listen, source: "env", lockedBy: "SLIPSTREAM_HTTP_LISTEN", isDefault: false }}
        onSave={saving()}
      />
      <SettingField
        field={{ ...compress, lockedBy: "SLIPSTREAM_LOG_COMPRESS" }}
        onSave={saving()}
      />
      <SettingField field={{ ...level, lockedBy: "SLIPSTREAM_LOG_LEVEL" }} onSave={saving()} />
    </>,
  );
  await expect.element(page.getByText("Set by SLIPSTREAM_HTTP_LISTEN")).toBeVisible();
  await expect.element(page.getByRole("textbox", { name: "Port" })).toBeDisabled();
  await expect.element(page.getByRole("switch", { name: "Compress rotated files" })).toBeDisabled();
  await expect.element(page.getByRole("combobox", { name: "Level" })).toBeDisabled();
  expect(page.getByRole("button", { name: "Reset to default" }).query()).toBeNull();
});

test("a count where 0 means no limit says so", async () => {
  const onSave = saving();
  renderKit(<SettingField field={daysKept} onSave={onSave} />);
  await expect
    .element(page.getByText("Default 30 days. Between 0 and 3650 days. 0 means no limit."))
    .toBeVisible();
  await expect.element(page.getByText("No limit", { exact: true })).toBeVisible();
  await userEvent.fill(page.getByRole("textbox", { name: "Days kept" }), "7");
  await expect.element(page.getByText("days", { exact: true })).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Reset to default" }));
  expect(onSave).toHaveBeenCalledExactlyOnceWith(null);
});

test("a switch saves the side it's turned to", async () => {
  const onSave = saving();
  renderKit(<SettingField field={compress} onSave={onSave} />);
  await expect.element(page.getByText("Default On.")).toBeVisible();
  await userEvent.click(page.getByRole("switch", { name: "Compress rotated files" }));
  expect(onSave).toHaveBeenCalledExactlyOnceWith(false);
});

test("a choice of four is a Select that saves the option picked", async () => {
  const onSave = saving();
  renderKit(<SettingField field={level} onSave={onSave} />);
  await userEvent.click(page.getByRole("combobox", { name: "Level" }));
  await userEvent.click(page.getByRole("option", { name: "Debug" }));
  expect(onSave).toHaveBeenCalledExactlyOnceWith("debug");
});

test("a row says a count of 0 in words", async () => {
  renderKit(<SettingField field={daysKept} onSave={saving()} variant="row" />);
  await expect.element(page.getByRole("button", { name: "Days kept: No limit" })).toBeVisible();
});
