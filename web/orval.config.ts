import { defineConfig } from "orval";

import { liftStreams } from "./orval/streams.ts";

const document = "./openapi.json";

export default defineConfig({
  api: {
    input: { target: document, override: { transformer: liftStreams } },
    output: {
      mode: "tags-split",
      target: "src/api/gen/query",
      schemas: "src/api/gen/model",
      client: "react-query",
      httpClient: "fetch",
      baseUrl: "/api/v1",
      clean: true,
      formatter: "oxfmt",
      tsconfig: "./tsconfig.json",
      mock: { generators: [{ type: "msw", delay: false, baseUrl: "*/api/v1" }] },
      override: {
        header: false,
        enumGenerationType: "union",
        fetch: { includeHttpResponseReturnType: false },
        mutator: { path: "src/api/fetch.ts", name: "apiFetch" },
        mock: { exactOptional: true },
        query: {
          useQuery: false,
          useSuspenseQuery: true,
          shouldExportKeys: true,
          signal: true,
          version: 5,
        },
      },
    },
  },
  apiZod: {
    input: { target: document, override: { transformer: liftStreams } },
    output: {
      mode: "tags-split",
      target: "src/api/gen/zod",
      client: "zod",
      fileExtension: ".zod.ts",
      clean: true,
      formatter: "oxfmt",
      tsconfig: "./tsconfig.json",
      override: {
        header: false,
        zod: {
          version: 4,
          generateReusableSchemas: true,
          generateDiscriminatedUnion: true,
          exactOptional: true,
        },
      },
    },
  },
});
