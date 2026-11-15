import path from "node:path";
import { fileURLToPath } from "node:url";

import copyWords from "./copy-words.json" with { type: "json" };
import houseBaseline from "./house-words-baseline.json" with { type: "json" };
import houseWords from "./house-words.json" with { type: "json" };

/**
 * @param {{ name: string; pattern: string; alone?: boolean; caseSensitive?: boolean }[]} list
 */
function compile(list) {
  return list.map(({ name, pattern, alone, caseSensitive }) => {
    const flags = caseSensitive === true ? "u" : "iu";
    return {
      name,
      re: new RegExp(pattern, flags),
      whole: alone === true ? new RegExp(`^(?:${pattern})$`, flags) : undefined,
    };
  });
}

const bannedCopy = compile(copyWords);

const repoRoot = fileURLToPath(new URL("../..", import.meta.url));

/** @type {Record<string, Record<string, number>>} */
const baseline = process.env.HOUSE_WORDS_BASELINE === "off" ? {} : houseBaseline;

const glossaryWords = compile(houseWords);

// A string with no space that starts lowercase, such as a route or a query
// key, is no copy. A one-word label such as "Cells" starts with a capital.
/** @param {string} text */
function isKey(text) {
  return !/^\p{Lu}|\s/u.test(text);
}

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

/**
 * @param {(node: unknown, text: string) => void} visit
 */
function stringVisitors(visit) {
  return {
    /** @param {{ value: unknown }} node */
    Literal(node) {
      if (typeof node.value === "string") {
        visit(node, node.value);
      }
    },
    /** @param {{ value: { raw: string; cooked: string | null } }} node */
    TemplateElement(node) {
      visit(node, node.value.cooked ?? node.value.raw);
    },
    /** @param {{ value: string }} node */
    JSXText(node) {
      visit(node, node.value);
    },
  };
}

// ESLint's no-restricted-syntax, which Oxlint does not ship: each option
// reports every node its selector matches with its message. copy-words
// reports strings that use the words and dashes WRITING.md bans, and
// house-words the glossary words rule 21 keeps out of copy beyond what
// house-words-baseline.json still allows, which the release lint shares.
export default {
  meta: { name: "slipstream" },
  rules: {
    "copy-words": {
      /**
       * @param {{ report: (problem: { node: unknown; message: string }) => void }} context
       */
      create(context) {
        return stringVisitors((node, text) => {
          reportCopy(context, node, text);
        });
      },
    },
    "house-words": {
      /**
       * @param {{
       *   physicalFilename: string;
       *   report: (problem: { node: unknown; message: string }) => void;
       * }} context
       */
      create(context) {
        const file = path.relative(repoRoot, context.physicalFilename).replaceAll("\\", "/");
        const allowed = baseline[file] ?? {};
        /** @type {Map<string, unknown[]>} */
        const hits = new Map(glossaryWords.map(({ name }) => [name, []]));
        return {
          ...stringVisitors((node, text) => {
            if (isKey(text)) {
              return;
            }
            const trimmed = text.trim();
            for (const { name, re, whole } of glossaryWords) {
              if (re.test(text) && whole?.test(trimmed) !== true) {
                hits.get(name)?.push(node);
              }
            }
          }),
          /** @param {unknown} program */
          "Program:exit"(program) {
            for (const [name, nodes] of hits) {
              const limit = allowed[name] ?? 0;
              if (nodes.length < limit) {
                const fix = nodes.length === 0 ? "Remove it" : `Lower it to ${nodes.length}`;
                context.report({
                  node: program,
                  message: `web/lint/house-words-baseline.json allows ${limit} "${name}" here and the file has ${nodes.length}. ${fix}`,
                });
              } else if (nodes.length > limit) {
                const over =
                  limit > 0
                    ? `. web/lint/house-words-baseline.json allows ${limit} here and the file has ${nodes.length}`
                    : "";
                for (const node of nodes) {
                  context.report({
                    node,
                    message: `"${name}" is a glossary word. Say what it means for the reader, per WRITING.md rule 21${over}`,
                  });
                }
              }
            }
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
