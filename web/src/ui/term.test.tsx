import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import glossary from "../../../GLOSSARY.md?raw";
import { renderKit } from "../test/render-kit";
import { Term, terms } from "./term";

/** Each glossary entry's `_In UI_` line, by the entry's name in lower case. */
function inUiLines(text: string): Map<string, string> {
  const lines = new Map<string, string>();
  for (const [, name, inUi] of text.matchAll(/^\*\*(.+?)\*\*:\n(?:.+\n)*?_In UI_: (.+)$/gmu)) {
    if (name !== undefined && inUi !== undefined) {
      lines.set(name.toLowerCase(), inUi);
    }
  }
  return lines;
}

test("every term names a glossary entry whose In UI line gives its words", () => {
  const lines = inUiLines(glossary);
  for (const [id, { words }] of Object.entries(terms)) {
    const inUi = lines.get(id);
    expect(inUi, `GLOSSARY.md has an entry for ${id}`).toBeDefined();
    expect(inUi, `${id} reaches the screen`).not.toMatch(/^none/u);
    expect(inUi, `${id}'s In UI line gives "${words}"`).toContain(`"${words}"`);
  }
});

test("a term opens its definition on hover", async () => {
  renderKit(
    <p>
      SlipStream is <Term id="reconciling" /> after a restore.
    </p>,
  );
  await userEvent.hover(page.getByRole("button", { name: "Reconciling" }));
  await expect
    .element(page.getByRole("dialog", { name: "Reconciling" }))
    .toHaveTextContent(terms.reconciling.definition);
});
