import { expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { ApiError } from "../api/errors";
import type {
  SettingChoiceField,
  SettingCountField,
  SettingDurationField,
  SettingWindowField,
} from "../api/gen/model";
import { renderKit } from "../test/render-kit";
import { SettingField } from "./setting-field";

const about = { source: "database", applies: "live" } as const;

const rss: SettingDurationField = {
  ...about,
  key: "tasks.rss.cadence",
  label: "RSS sync",
  kind: "duration",
  isDefault: true,
  value: 900,
  default: 900,
  min: 600,
  max: 7200,
  step: 60,
  floor: "Private trackers ban for hammering.",
};

const housekeeping: SettingDurationField = {
  ...about,
  key: "tasks.housekeeping.cadence",
  label: "Housekeeping",
  kind: "duration",
  isDefault: false,
  value: 172_800,
  default: 86_400,
  min: 86_400,
  max: 604_800,
  step: 86_400,
};

const runsKept: SettingCountField = {
  ...about,
  key: "tasks.runsKept",
  label: "Runs kept",
  kind: "count",
  isDefault: true,
  value: 50,
  default: 50,
  min: 1,
  max: 1000,
  unit: "runs",
};

const overdue: SettingChoiceField = {
  ...about,
  key: "tasks.housekeeping.overdue",
  label: "When overdue",
  kind: "choice",
  isDefault: true,
  defaultFrom: "Scheduled tasks",
  value: "catch-up",
  default: "catch-up",
  options: [
    { value: "catch-up", label: "Catch up" },
    { value: "wait", label: "Wait" },
  ],
};

const maintenance: SettingWindowField = {
  ...about,
  key: "tasks.window",
  label: "Maintenance window",
  kind: "window",
  isDefault: true,
  value: { start: "04:00", end: "06:00" },
  default: { start: "04:00", end: "06:00" },
  zone: "Europe/London",
};

function saving() {
  return vi.fn<(value: unknown) => Promise<unknown>>(() => Promise.resolve());
}

const outOfRange = (detail: string) =>
  new ApiError({
    status: 422,
    code: "setting_out_of_range",
    title: "The setting is out of range",
    detail,
  });

test("a duration shows its default, its range and its floor", async () => {
  renderKit(<SettingField field={rss} onSave={saving()} />);
  await expect.element(page.getByText("Default 15 min. Between 10 min and 2 h.")).toBeVisible();
  await expect
    .element(page.getByText("No faster than 10 min. Private trackers ban for hammering."))
    .toBeVisible();
  await expect.element(page.getByRole("textbox", { name: "RSS sync" })).toHaveValue("15");
  await expect.element(page.getByRole("radio", { name: "min" })).toBeChecked();
  expect(page.getByRole("button", { name: "Reset to default" }).query()).toBeNull();
});

test("a duration saves seconds in the unit chosen", async () => {
  const onSave = saving();
  renderKit(<SettingField field={rss} onSave={onSave} />);
  await userEvent.fill(page.getByRole("textbox", { name: "RSS sync" }), "30");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  expect(onSave).toHaveBeenLastCalledWith(1800);

  await userEvent.click(page.getByRole("radio", { name: "h" }));
  await userEvent.fill(page.getByRole("textbox", { name: "RSS sync" }), "1.5");
  await userEvent.keyboard("{Enter}");
  expect(onSave).toHaveBeenLastCalledWith(5400);
});

test("a duration below its floor is refused with the floor before any request", async () => {
  const onSave = saving();
  renderKit(<SettingField field={rss} onSave={onSave} />);
  const input = page.getByRole("textbox", { name: "RSS sync" });
  await userEvent.fill(input, "5");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect
    .element(page.getByRole("alert"))
    .toHaveTextContent("No faster than 10 min. Private trackers ban for hammering.");
  await expect.element(input).toHaveAttribute("aria-invalid", "true");
  expect(onSave).not.toHaveBeenCalled();

  await userEvent.fill(input, "3");
  await userEvent.click(page.getByRole("radio", { name: "h" }));
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByRole("alert")).toHaveTextContent("Between 10 min and 2 h.");
  expect(onSave).not.toHaveBeenCalled();
});

test("a duration stepped in days refuses part of a day and has no unit picker", async () => {
  const onSave = saving();
  renderKit(<SettingField field={housekeeping} onSave={onSave} />);
  const input = page.getByRole("textbox", { name: "Housekeeping" });
  await expect.element(input).toHaveValue("2");
  expect(page.getByRole("radiogroup").query()).toBeNull();
  await userEvent.fill(input, "1.5");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByRole("alert")).toHaveTextContent("In whole days.");
  expect(onSave).not.toHaveBeenCalled();
});

test("Reset to default sends null and shows only off the default", async () => {
  const onSave = saving();
  renderKit(<SettingField field={housekeeping} onSave={onSave} />);
  await expect.element(page.getByText("Default 1 day. Between 1 day and 7 days.")).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Reset to default" }));
  expect(onSave).toHaveBeenCalledExactlyOnceWith(null);
});

test("the server's refusal shows as the field's error", async () => {
  const onSave = vi.fn<(value: unknown) => Promise<unknown>>(() =>
    Promise.reject(outOfRange("No faster than 1 h. Indexers rate-limit searches.")),
  );
  renderKit(<SettingField field={rss} onSave={onSave} />);
  await userEvent.fill(page.getByRole("textbox", { name: "RSS sync" }), "20");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  expect(onSave).toHaveBeenCalledExactlyOnceWith(1200);
  await expect
    .element(page.getByRole("alert"))
    .toHaveTextContent("No faster than 1 h. Indexers rate-limit searches.");

  await userEvent.type(page.getByRole("textbox", { name: "RSS sync" }), "5");
  await expect.element(page.getByRole("alert")).not.toBeInTheDocument();
});

test("a count saves the number typed and refuses one outside its range", async () => {
  const onSave = saving();
  renderKit(<SettingField field={runsKept} onSave={onSave} />);
  await expect.element(page.getByText("Default 50 runs. Between 1 and 1000 runs.")).toBeVisible();
  const input = page.getByRole("textbox", { name: "Runs kept" });
  await userEvent.fill(input, "0");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByRole("alert")).toHaveTextContent("Between 1 and 1000 runs.");
  expect(onSave).not.toHaveBeenCalled();

  await userEvent.fill(input, "200");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  expect(onSave).toHaveBeenCalledExactlyOnceWith(200);
});

test("a choice names where its default comes from and saves the option chosen", async () => {
  const onSave = saving();
  renderKit(<SettingField field={overdue} onSave={onSave} />);
  await expect.element(page.getByText("Default Catch up, from Scheduled tasks.")).toBeVisible();
  await userEvent.click(page.getByRole("radio", { name: "Wait" }));
  expect(onSave).toHaveBeenCalledExactlyOnceWith("wait");
});

test("a window shows its zone and saves both times", async () => {
  const onSave = saving();
  renderKit(<SettingField field={maintenance} onSave={onSave} />);
  await expect
    .element(page.getByRole("group", { name: "Maintenance window" }).getByText("Europe/London"))
    .toBeVisible();
  await userEvent.fill(page.getByLabelText("From"), "03:00");
  await userEvent.fill(page.getByLabelText("to", { exact: true }), "03:00");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect
    .element(page.getByRole("alert"))
    .toHaveTextContent("The window needs an end after its start.");
  expect(onSave).not.toHaveBeenCalled();

  await userEvent.fill(page.getByLabelText("From"), "01:00");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  expect(onSave).toHaveBeenCalledExactlyOnceWith({ start: "01:00", end: "03:00" });
});

test("a row shows the value in words and opens the full field, which closes once saved", async () => {
  const onSave = saving();
  renderKit(
    <SettingField
      field={housekeeping}
      onSave={onSave}
      variant="row"
      words="Every 2 days in the Maintenance window"
    />,
  );
  const row = page.getByRole("button", {
    name: "Housekeeping: Every 2 days in the Maintenance window",
  });
  await userEvent.click(row);
  const popover = page.getByRole("dialog", { name: "Housekeeping" });
  await userEvent.fill(popover.getByRole("textbox", { name: "Housekeeping" }), "3");
  await userEvent.click(popover.getByRole("button", { name: "Save" }));
  expect(onSave).toHaveBeenCalledExactlyOnceWith(259_200);
  await expect.element(popover).not.toBeInTheDocument();
});

test("a row without words says the value, and opens a sheet on phone", async () => {
  renderKit(<SettingField field={runsKept} onSave={saving()} variant="row" />, {
    shell: "phone",
  });
  await userEvent.click(page.getByRole("button", { name: "Runs kept: 50 runs" }));
  const sheet = page.getByRole("dialog", { name: "Runs kept" });
  await expect.element(sheet.getByRole("textbox", { name: "Runs kept" })).toHaveValue("50");
  await userEvent.click(sheet.getByRole("button", { name: "Cancel" }));
  await expect.element(sheet).not.toBeInTheDocument();
});
