import { useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { FileField } from "./file-field";

function Picker() {
  const [text, setText] = useState("");
  return (
    <>
      <FileField
        label="Definition file"
        accept=".yml"
        description="A YAML file."
        onFile={async (file) => {
          setText(await file.text());
        }}
      />
      <output>{text}</output>
    </>
  );
}

test("a file field hands over the chosen file and shows its name", async () => {
  renderKit(<Picker />);
  const group = page.getByRole("group", { name: "Definition file" });
  await expect
    .element(group.getByRole("button", { name: "Choose a file" }))
    .toHaveAccessibleDescription("A YAML file.");
  await userEvent.upload(
    page.getByLabelText("Definition file"),
    new File(["id: meadow"], "meadow.yml", { type: "application/yaml" }),
  );
  await expect.element(group.getByText("meadow.yml")).toBeVisible();
  await expect.element(page.getByRole("status")).toHaveTextContent("id: meadow");
});

test("a file dropped on the field is handed over as a chosen one is", async () => {
  renderKit(<Picker />);
  const group = page.getByRole("group", { name: "Definition file" });
  await expect.element(group).toBeVisible();
  const drop = new DataTransfer();
  drop.items.add(new File(["id: orchard"], "orchard.yml", { type: "application/yaml" }));
  group.element().dispatchEvent(new DragEvent("drop", { dataTransfer: drop, bubbles: true }));
  await expect.element(group.getByText("orchard.yml")).toBeVisible();
  await expect.element(page.getByRole("status")).toHaveTextContent("id: orchard");
});
