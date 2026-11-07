import { useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { useAppForm } from "./form";

function KeyForm({ saved }: { saved: boolean }) {
  const [sent, setSent] = useState<string>();
  const form = useAppForm({
    defaultValues: { apiKey: "" },
    onSubmit: ({ value }) => {
      setSent(value.apiKey === "" ? "nothing" : value.apiKey);
    },
  });
  return (
    <form
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <form.AppForm>
        <form.AppField name="apiKey">
          {(field) => <field.SecretField label="API key" saved={saved} shown="indexer.test" />}
        </form.AppField>
        <form.SubmitButton label="Save" />
        {sent === undefined ? null : <output>Sent {sent}</output>}
      </form.AppForm>
    </form>
  );
}

test("a saved secret reads Saved and is never sent until it is replaced", async () => {
  renderKit(<KeyForm saved />);
  const field = page.getByRole("group", { name: "API key" });
  await expect.element(field.getByText("Saved")).toBeVisible();
  await expect.element(field.getByText("indexer.test")).toBeVisible();
  expect(document.querySelector("input")).toBeNull();
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Sent nothing")).toBeVisible();

  await userEvent.click(page.getByRole("button", { name: "Replace API key" }));
  const input = page.getByLabelText("API key", { exact: true });
  await expect.element(input).toHaveFocus();
  await userEvent.fill(input, "typed");
  await userEvent.click(page.getByRole("button", { name: "Cancel" }));
  await expect
    .element(page.getByRole("group", { name: "API key" }).getByText("Saved"))
    .toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Sent nothing")).toBeVisible();

  await userEvent.click(page.getByRole("button", { name: "Replace API key" }));
  await userEvent.fill(page.getByLabelText("API key", { exact: true }), "fresh-key");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Sent fresh-key")).toBeVisible();
});

test("a secret not saved yet is an input from the start", async () => {
  renderKit(<KeyForm saved={false} />);
  await expect.element(page.getByLabelText("API key", { exact: true })).toBeVisible();
  expect(page.getByRole("button", { name: "Replace API key" }).query()).toBeNull();
  expect(page.getByRole("button", { name: "Cancel" }).query()).toBeNull();
});
