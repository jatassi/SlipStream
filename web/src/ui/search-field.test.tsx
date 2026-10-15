import { useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { SearchField } from "./search-field";
import type { ShellKind } from "./shell-context";

function Find() {
  const [query, setQuery] = useState("");
  return (
    <>
      <SearchField label="Search the Library" value={query} onValueChange={setQuery} />
      <p>Query: {query}</p>
    </>
  );
}

test.each([
  ["desktop", "32px", "13px"],
  ["phone", "44px", "17px"],
] satisfies [ShellKind, string, string][])(
  "in the %s shell a search capsule is %s tall with %s text and takes typing",
  async (shell, height, size) => {
    renderKit(<Find />, { shell });
    const box = page.getByRole("searchbox", { name: "Search the Library" });
    await expect.element(box).toBeVisible();
    const capsule = getComputedStyle(box.element());
    expect([capsule.height, capsule.borderRadius, capsule.fontSize]).toEqual([
      height,
      "999px",
      size,
    ]);
    await userEvent.type(box, "dune");
    await expect.element(page.getByText("Query: dune")).toBeVisible();
  },
);
