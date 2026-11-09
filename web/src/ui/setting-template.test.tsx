import { expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { ApiError } from "../api/errors";
import type { SettingTemplateField } from "../api/gen/model";
import { renderKit } from "../test/render-kit";
import { SettingField } from "./setting-field";

const colonCustom: SettingTemplateField = {
  source: "database",
  applies: "live",
  pendingRestart: false,
  key: "show.naming.colonCustom",
  label: "Custom colon",
  kind: "template",
  isDefault: true,
  value: " - ",
  default: " - ",
  field: "colonCustom",
  plain: true,
};

const refused = new ApiError({
  status: 422,
  code: "setting_out_of_range",
  title: "The value is outside the setting's range",
  detail: "This text can't hold a token. Remove each { and }.",
});

test("a template saves the text as typed and shows its default", async () => {
  const onSave = vi.fn<(value: unknown) => Promise<unknown>>(() => Promise.resolve());
  renderKit(<SettingField field={colonCustom} onSave={onSave} />);
  await expect.element(page.getByText('Default " - ".')).toBeVisible();
  const text = page.getByRole("textbox", { name: "Custom colon" });
  await expect.element(text).toHaveValue(" - ");
  expect(page.getByRole("button", { name: "Save" }).query()).toBeNull();

  await userEvent.fill(text, " ~ ");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  expect(onSave).toHaveBeenCalledExactlyOnceWith(" ~ ");
});

test("the owner's refusal shows under the template until it is edited", async () => {
  const onSave = vi.fn<(value: unknown) => Promise<unknown>>(() => Promise.reject(refused));
  renderKit(<SettingField field={colonCustom} onSave={onSave} />);
  const text = page.getByRole("textbox", { name: "Custom colon" });
  await userEvent.fill(text, "{Title}");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  expect(onSave).toHaveBeenCalledExactlyOnceWith("{Title}");
  await expect
    .element(page.getByRole("alert"))
    .toHaveTextContent("This text can't hold a token. Remove each { and }.");
  await expect.element(text).toHaveAttribute("aria-invalid", "true");

  await userEvent.type(text, "x");
  await expect.element(page.getByRole("alert")).not.toBeInTheDocument();
});
