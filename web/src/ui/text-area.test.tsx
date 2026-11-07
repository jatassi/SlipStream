import { useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { useAppForm } from "./form";

function NotesForm() {
  const [sent, setSent] = useState<string>();
  const form = useAppForm({
    defaultValues: { notes: "" },
    onSubmit: ({ value }) => {
      setSent(value.notes);
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
        <form.AppField
          name="notes"
          validators={{ onSubmit: ({ value }) => (value === "" ? "Write the notes." : undefined) }}
        >
          {(field) => <field.TextArea label="Notes" rows={4} description="Kept as typed." />}
        </form.AppField>
        <form.SubmitButton label="Save" />
        {sent === undefined ? null : <output>{sent}</output>}
      </form.AppForm>
    </form>
  );
}

test("a text area keeps several lines, and shows its error in place of its description", async () => {
  renderKit(<NotesForm />);
  const area = page.getByRole("textbox", { name: "Notes" });
  await expect.element(area).toHaveAccessibleDescription("Kept as typed.");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Write the notes.")).toBeVisible();
  await userEvent.fill(area, "first\nsecond");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByRole("status")).toHaveTextContent("first second");
});
