const options = {
  type: "array",
  items: {
    type: "object",
    properties: { selector: { type: "string" }, message: { type: "string" } },
    required: ["selector", "message"],
    additionalProperties: false,
  },
};

// ESLint's no-restricted-syntax, which Oxlint does not ship: each option
// reports every node its selector matches with its message.
export default {
  meta: { name: "slipstream" },
  rules: {
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
