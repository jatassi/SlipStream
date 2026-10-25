import copyWords from "./copy-words.json" with { type: "json" };

const bannedCopy = copyWords.map(({ name, pattern, alone }) => ({
  name,
  re: new RegExp(pattern, "iu"),
  whole: alone === true ? new RegExp(`^(?:${pattern})$`, "iu") : undefined,
}));

const options = {
  type: "array",
  items: {
    type: "object",
    properties: { selector: { type: "string" }, message: { type: "string" } },
    required: ["selector", "message"],
    additionalProperties: false,
  },
};

/**
 * @param {{ report: (problem: { node: unknown; message: string }) => void }} context
 * @param {unknown} node
 * @param {string} text
 */
function reportCopy(context, node, text) {
  const trimmed = text.trim();
  for (const { name, re, whole } of bannedCopy) {
    if (re.test(text) && whole?.test(trimmed) !== true) {
      context.report({
        node,
        message: `"${name}" is banned in user-facing strings. Rewrite it per WRITING.md`,
      });
    }
  }
}

// ESLint's no-restricted-syntax, which Oxlint does not ship: each option
// reports every node its selector matches with its message. copy-words
// reports strings that use the words and dashes WRITING.md bans.
export default {
  meta: { name: "slipstream" },
  rules: {
    "copy-words": {
      /**
       * @param {{ report: (problem: { node: unknown; message: string }) => void }} context
       */
      create(context) {
        return {
          /** @param {{ value: unknown }} node */
          Literal(node) {
            if (typeof node.value === "string") {
              reportCopy(context, node, node.value);
            }
          },
          /** @param {{ value: { raw: string; cooked: string | null } }} node */
          TemplateElement(node) {
            reportCopy(context, node, node.value.cooked ?? node.value.raw);
          },
          /** @param {{ value: string }} node */
          JSXText(node) {
            reportCopy(context, node, node.value);
          },
        };
      },
    },
    "no-restricted-syntax": {
      meta: { schema: options },
      /**
       * @param {{
       *   options: { selector: string; message: string }[];
       *   report: (problem: { node: unknown; message: string }) => void;
       * }} context
       */
      create(context) {
        return Object.fromEntries(
          context.options.map(({ selector, message }) => [
            selector,
            /** @param {unknown} node */
            (node) => {
              context.report({ node, message });
            },
          ]),
        );
      },
    },
  },
};
