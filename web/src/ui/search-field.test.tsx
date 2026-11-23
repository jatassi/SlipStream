import { useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { SearchField } from "./search-field";

function Find() {
  const [query, setQuery] = useState("");
  return (
    <>
      <SearchField label="Search the Library" value={query} onValueChange={setQuery} />
      <p>Query: {query}</p>
    </>
  );
}

test("a search field takes typing", async () => {
  renderKit(<Find />);
  await userEvent.type(page.getByRole("searchbox", { name: "Search the Library" }), "dune");
  await expect.element(page.getByText("Query: dune")).toBeVisible();
});
