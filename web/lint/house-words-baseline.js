// Rewrites house-words-baseline.json to the matches the Go and web sides of
// house-words find today. Run it from web/ with `bun lint/house-words-baseline.js`.
import { spawnSync } from "node:child_process";
import { writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const web = fileURLToPath(new URL("..", import.meta.url));
const repo = fileURLToPath(new URL("../..", import.meta.url));
const env = { ...process.env, HOUSE_WORDS_BASELINE: "off" };

/**
 * @param {string} cmd
 * @param {string[]} args
 * @param {string} cwd
 */
function output(cmd, args, cwd) {
  const run = spawnSync(cmd, args, {
    cwd,
    env,
    encoding: "utf8",
    maxBuffer: 1 << 28,
    stdio: ["ignore", "pipe", "inherit"],
  });
  if (run.error !== undefined) {
    throw run.error;
  }
  return run.stdout;
}

/** @type {Record<string, Record<string, number>>} */
const counts = {};

/**
 * @param {string} file
 * @param {string} message
 */
function count(file, message) {
  const name = /^"([^"]+)" is a glossary word/u.exec(message)?.[1];
  if (name === undefined) {
    throw new Error(`${file}: house-words reported "${message}", which names no word`);
  }
  const words = (counts[file] ??= {});
  words[name] = (words[name] ?? 0) + 1;
}

for (const line of output(
  "go",
  ["run", "./cmd/release", "lint", "-rule", "house-words"],
  repo,
).split("\n")) {
  const hit = /^(.+?):\d+: \[house-words\] (.+)$/u.exec(line);
  if (hit?.[1] !== undefined && hit[2] !== undefined) {
    count(hit[1], hit[2]);
  }
}

const oxlint = output(
  process.execPath,
  ["node_modules/oxlint/bin/oxlint", "--format", "unix"],
  web,
);
for (const line of oxlint.split("\n")) {
  const hit = /^(.+?):\d+:\d+: (.+) \[Error\/slipstream\(house-words\)\]$/u.exec(line);
  if (hit?.[1] !== undefined && hit[2] !== undefined) {
    count(`web/${hit[1].replaceAll("\\", "/")}`, hit[2]);
  }
}

const sorted = Object.fromEntries(
  Object.keys(counts)
    .toSorted()
    .map((file) => [
      file,
      Object.fromEntries(
        Object.entries(counts[file] ?? {}).toSorted(([a], [b]) => a.localeCompare(b)),
      ),
    ]),
);
writeFileSync(
  new URL("house-words-baseline.json", import.meta.url),
  `${JSON.stringify(sorted, null, 2)}\n`,
);
