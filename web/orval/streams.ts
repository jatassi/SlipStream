import type { OpenApiDocument } from "@orval/core";

type Schema = Record<string, unknown>;

// Each event stream's union, by operation. A stream the document gains
// must be named here before the client generates.
const unions: Record<string, string> = {
  streamEvents: "LiveEvent",
  streamNodeLogs: "LogEvent",
};

const expected =
  "a 200 text/event-stream schema as an array whose items.oneOf holds {event: {const: <name>}, data: <schema>} objects";

const isObject = (value: unknown): value is Schema =>
  typeof value === "object" && value !== null && !Array.isArray(value);

const branchName = (event: string) =>
  event
    .split(/[.-]/u)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join("");

function okContent(operation: Schema): Schema | undefined {
  const responses = operation["responses"];
  const ok = isObject(responses) ? responses["200"] : undefined;
  const content = isObject(ok) ? ok["content"] : undefined;
  return isObject(content) ? content : undefined;
}

function events(id: string, schema: unknown): [string, unknown][] {
  const items = isObject(schema) ? schema["items"] : undefined;
  const oneOf = isObject(items) ? items["oneOf"] : undefined;
  if (
    !isObject(schema) ||
    schema["type"] !== "array" ||
    !Array.isArray(oneOf) ||
    oneOf.length === 0
  ) {
    throw new Error(`liftStreams: ${id} has no ${expected}`);
  }
  return oneOf
    .map((branch: unknown): [string, unknown] => {
      const properties = isObject(branch) ? branch["properties"] : undefined;
      const event = isObject(properties) ? properties["event"] : undefined;
      const name = isObject(event) ? event["const"] : undefined;
      if (typeof name !== "string" || !isObject(properties) || !isObject(properties["data"])) {
        throw new Error(
          `liftStreams: a branch of ${id} has no event const or data; expected ${expected}`,
        );
      }
      return [name, properties["data"]];
    })
    .toSorted(([a], [b]) => a.localeCompare(b));
}

function union(name: string, lifted: [string, unknown][]): Record<string, Schema> {
  const schemas: Record<string, Schema> = {};
  const mapping: Record<string, string> = {};
  for (const [event, data] of lifted) {
    const branch = `${name}_${branchName(event)}`;
    schemas[branch] = {
      type: "object",
      additionalProperties: false,
      properties: { event: { type: "string", const: event }, data },
      required: ["event", "data"],
    };
    mapping[event] = `#/components/schemas/${branch}`;
  }
  schemas[name] = {
    oneOf: Object.values(mapping).map(($ref) => ({ $ref })),
    discriminator: { propertyName: "event", mapping },
  };
  return schemas;
}

// liftStreams turns each event stream's events into a union schema the
// client parses with, and drops from the generated client every GET that
// answers no JSON: the streams, which live/ opens, and the downloads, which
// the browser saves through api/urls.ts.
export function liftStreams(spec: OpenApiDocument): OpenApiDocument {
  const schemas: Record<string, Schema> = {};
  const paths: NonNullable<OpenApiDocument["paths"]> = {};
  for (const [path, item] of Object.entries(spec.paths ?? {})) {
    const operation: unknown = item.get;
    const content = isObject(operation) ? okContent(operation) : undefined;
    if (!isObject(operation) || content === undefined || "application/json" in content) {
      paths[path] = item;
      continue;
    }
    const id = String(operation["operationId"]);
    const stream = content["text/event-stream"];
    if (stream !== undefined) {
      const name = unions[id];
      if (name === undefined) {
        throw new Error(
          `liftStreams: name the event union of the stream ${id} in orval/streams.ts`,
        );
      }
      Object.assign(
        schemas,
        union(name, events(id, isObject(stream) ? stream["schema"] : undefined)),
      );
    }
    const { get: _get, ...rest } = item;
    if (Object.keys(rest).some((key) => key !== "parameters")) {
      paths[path] = rest;
    }
  }
  const missing = Object.entries(unions).filter(([, name]) => !(name in schemas));
  if (missing.length > 0) {
    throw new Error(
      `liftStreams: the document has no stream ${missing.map(([id]) => id).join(", ")}; expected ${expected}`,
    );
  }
  return {
    ...spec,
    paths,
    components: { ...spec.components, schemas: { ...spec.components?.schemas, ...schemas } },
  };
}
