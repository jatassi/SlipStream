import { expect, test } from "vitest";

import openapi from "../../openapi.json" with { type: "json" };
import { nodeLogsArchiveUrl, nodeLogsStreamUrl } from "./urls";

const documented = (url: string) =>
  url
    .replace(/^\/api\/v1/u, "")
    .replace(/\?.*$/u, "")
    .replace("/nodes/ID/", "/nodes/{id}/");

test.each([
  ["nodeLogsArchiveUrl", nodeLogsArchiveUrl("ID"), "get"],
  ["nodeLogsStreamUrl", nodeLogsStreamUrl("ID", 3), "get"],
])("%s names a documented operation", (_name, url, method) => {
  const paths: Record<string, Record<string, unknown>> = openapi.paths;
  expect(paths[documented(url)]?.[method]).toBeDefined();
});

test("a stream resumes after a record", () => {
  expect(nodeLogsStreamUrl("0b0b", 42)).toBe("/api/v1/nodes/0b0b/logs/stream?after=42");
  expect(nodeLogsStreamUrl("0b0b")).toBe("/api/v1/nodes/0b0b/logs/stream");
});
