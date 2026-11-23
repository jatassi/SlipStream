import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { useAppForm } from "./form";

function NameForm() {
  const form = useAppForm({
    defaultValues: { name: "", pin: "" },
    validators: {
      onChange: ({ value }) =>
        value.name === "x" ? { fields: { name: "Use at least two letters" } } : undefined,
    },
  });
  return (
    <form.AppForm>
      <form.AppField name="name">{(field) => <field.TextField label="Name" />}</form.AppField>
      <form.AppField name="pin">
        {(field) => <field.PinField label="PIN" autoComplete="new-password" />}
      </form.AppField>
    </form.AppForm>
  );
}

test("a field takes typing and shows its error beside the input", async () => {
  renderKit(<NameForm />);
  await userEvent.type(page.getByRole("textbox", { name: "Name" }), "x");
  await expect.element(page.getByText("Use at least two letters")).toBeVisible();
  await userEvent.type(page.getByRole("textbox", { name: "Name" }), "y");
  await expect.element(page.getByText("Use at least two letters")).not.toBeInTheDocument();
});

test("the PIN moves to its next digit as it is typed", async () => {
  renderKit(<NameForm />);
  await page.getByRole("textbox", { name: "PIN", exact: true }).click();
  await userEvent.keyboard("12");
  await expect.element(page.getByRole("textbox", { name: "PIN digit 3 of 4" })).toHaveFocus();
});
