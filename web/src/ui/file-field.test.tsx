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
