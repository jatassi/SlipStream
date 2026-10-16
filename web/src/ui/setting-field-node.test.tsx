import { expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { ApiError } from "../api/errors";
import type {
  SettingChoiceField,
  SettingCountField,
  SettingPortField,
  SettingSwitchField,
} from "../api/gen/model";
import { renderKit } from "../test/render-kit";
import { SettingField } from "./setting-field";
import type { ShellKind } from "./shell-context";

const about = { source: "config.yaml", applies: "live", pendingRestart: false } as const;

const port: SettingPortField = {
  ...about,
  key: "http.listen",
  label: "Port",
  kind: "port",
  applies: "restart",
  isDefault: true,
  value: 8484,
  running: 8484,
  default: 8484,
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

const inEachShell = test.each(["desktop", "phone"] satisfies ShellKind[]);

inEachShell("%s: each Node kind is a control named for its setting", async (shell) => {
  renderKit(
    <>
      <SettingField field={port} onSave={saving()} />
      <SettingField field={daysKept} onSave={saving()} />
      <SettingField field={compress} onSave={saving()} />
      <SettingField field={level} onSave={saving()} />
    </>,
    { shell },
  );
  await expect.element(page.getByRole("textbox", { name: "Port" })).toHaveValue("8484");
  await expect.element(page.getByRole("textbox", { name: "Days kept" })).toHaveValue("0");
  await expect.element(page.getByRole("switch", { name: "Compress rotated files" })).toBeChecked();
  await expect.element(page.getByRole("combobox", { name: "Level" })).toHaveTextContent("Info");
});

inEachShell("%s: a port shows its default and refuses one out of range", async (shell) => {
  const onSave = saving();
  renderKit(<SettingField field={port} onSave={onSave} />, { shell });
  await expect.element(page.getByText("Default 8484.")).toBeVisible();
  await expect.element(page.getByText("Applies after restart.")).toBeVisible();
  const input = page.getByRole("textbox", { name: "Port" });
  await userEvent.fill(input, "70000");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByRole("alert")).toHaveTextContent("Between 1 and 65535.");
  expect(onSave).not.toHaveBeenCalled();

  await userEvent.fill(input, "9000");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  expect(onSave).toHaveBeenCalledExactlyOnceWith(9000);
});

inEachShell("%s: a port the server refuses shows the problem's sentence", async (shell) => {
  const onSave = vi.fn<(value: unknown) => Promise<unknown>>(() =>
    Promise.reject(
      new ApiError({
        status: 422,
        code: "port_used_by_node",
        title: "The port is taken by this Node",
        detail: "8485 is this Node's link port.",
      }),
    ),
  );
  renderKit(<SettingField field={port} onSave={onSave} />, { shell });
  await userEvent.fill(page.getByRole("textbox", { name: "Port" }), "8485");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByRole("alert")).toHaveTextContent("8485 is this Node's link port.");
});

inEachShell("%s: a key set by the environment is locked and names its variable", async (shell) => {
  renderKit(
    <>
      <SettingField
        field={{ ...port, source: "env", lockedBy: "SLIPSTREAM_HTTP_LISTEN", isDefault: false }}
        onSave={saving()}
      />
      <SettingField
        field={{ ...compress, lockedBy: "SLIPSTREAM_LOG_COMPRESS" }}
        onSave={saving()}
      />
      <SettingField field={{ ...level, lockedBy: "SLIPSTREAM_LOG_LEVEL" }} onSave={saving()} />
    </>,
    { shell },
  );
  await expect.element(page.getByText("Set by SLIPSTREAM_HTTP_LISTEN")).toBeVisible();
  await expect.element(page.getByRole("textbox", { name: "Port" })).toBeDisabled();
  await expect.element(page.getByRole("switch", { name: "Compress rotated files" })).toBeDisabled();
  await expect.element(page.getByRole("combobox", { name: "Level" })).toBeDisabled();
  expect(page.getByRole("button", { name: "Reset to default" }).query()).toBeNull();
});

inEachShell("%s: a count where 0 means no limit says so", async (shell) => {
  const onSave = saving();
  renderKit(<SettingField field={daysKept} onSave={onSave} />, { shell });
  await expect
    .element(page.getByText("Default 30 days. Between 0 and 3650 days. 0 means no limit."))
    .toBeVisible();
  await expect.element(page.getByText("No limit", { exact: true })).toBeVisible();
  await userEvent.fill(page.getByRole("textbox", { name: "Days kept" }), "7");
  await expect.element(page.getByText("days", { exact: true })).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Reset to default" }));
  expect(onSave).toHaveBeenCalledExactlyOnceWith(null);
});

inEachShell("%s: a switch saves the side it's turned to", async (shell) => {
  const onSave = saving();
  renderKit(<SettingField field={compress} onSave={onSave} />, { shell });
  await expect.element(page.getByText("Default On.")).toBeVisible();
  await userEvent.click(page.getByRole("switch", { name: "Compress rotated files" }));
  expect(onSave).toHaveBeenCalledExactlyOnceWith(false);
});

inEachShell("%s: a choice of four is a Select that saves the option picked", async (shell) => {
  const onSave = saving();
  renderKit(<SettingField field={level} onSave={onSave} />, { shell });
  await userEvent.click(page.getByRole("combobox", { name: "Level" }));
  await userEvent.click(page.getByRole("option", { name: "Debug" }));
  expect(onSave).toHaveBeenCalledExactlyOnceWith("debug");
});

inEachShell("%s: a row says a count of 0 in words", async (shell) => {
  renderKit(<SettingField field={daysKept} onSave={saving()} variant="row" />, { shell });
  await expect.element(page.getByRole("button", { name: "Days kept: No limit" })).toBeVisible();
});
