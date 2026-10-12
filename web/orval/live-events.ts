import type { OpenApiDocument } from "@orval/core";

type Schema = Record<string, unknown>;

const expected =
  "the streamEvents operation's 200 text/event-stream schema as an array whose items.oneOf holds {event: {const: <name>}, data: <schema>} objects";

const isObject = (value: unknown): value is Schema =>
  typeof value === "object" && value !== null && !Array.isArray(value);

const branchName = (event: string) =>
  event
    .split(/[.-]/u)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join("");

function streamOperation(spec: OpenApiDocument): { path: string; schema: Schema } {
  for (const [path, item] of Object.entries(spec.paths ?? {})) {
    const operation: unknown = item.get;
    if (!isObject(operation) || operation["operationId"] !== "streamEvents") {
      continue;
    }
    const responses = operation["responses"];
    const ok = isObject(responses) ? responses["200"] : undefined;
    const content = isObject(ok) ? ok["content"] : undefined;
    const media = isObject(content) ? content["text/event-stream"] : undefined;
    const schema = isObject(media) ? media["schema"] : undefined;
    if (!isObject(schema)) {
      throw new Error(`liftLiveEvents: ${path} has no ${expected}`);
    }
    return { path, schema };
  }
  throw new Error(
    `liftLiveEvents: the document has no streamEvents operation; expected ${expected}`,
  );
}

function events(schema: Schema): [string, unknown][] {
  const items = schema["items"];
  const oneOf = isObject(items) ? items["oneOf"] : undefined;
  if (schema["type"] !== "array" || !Array.isArray(oneOf) || oneOf.length === 0) {
    throw new Error(`liftLiveEvents: expected ${expected}`);
  }
  return oneOf
    .map((branch: unknown): [string, unknown] => {
      const properties = isObject(branch) ? branch["properties"] : undefined;
      const event = isObject(properties) ? properties["event"] : undefined;
      const name = isObject(event) ? event["const"] : undefined;
      if (typeof name !== "string" || !isObject(properties) || !isObject(properties["data"])) {
        throw new Error(
          `liftLiveEvents: a branch has no event const or data; expected ${expected}`,
        );
      }
      return [name, properties["data"]];
    })
    .toSorted(([a], [b]) => a.localeCompare(b));
}

export function liftLiveEvents(spec: OpenApiDocument): OpenApiDocument {
  const { path, schema } = streamOperation(spec);
  const branches: Record<string, Schema> = {};
  const mapping: Record<string, string> = {};
  for (const [event, data] of events(schema)) {
    const name = `LiveEvent_${branchName(event)}`;
    branches[name] = {
      type: "object",
      additionalProperties: false,
      properties: { event: { type: "string", const: event }, data },
      required: ["event", "data"],
    };
    mapping[event] = `#/components/schemas/${name}`;
  }
  const { [path]: _stream, ...paths } = spec.paths ?? {};
  return {
    ...spec,
    paths,
    components: {
      ...spec.components,
      schemas: {
        ...spec.components?.schemas,
        ...branches,
        LiveEvent: {
          oneOf: Object.values(mapping).map(($ref) => ({ $ref })),
          discriminator: { propertyName: "event", mapping },
        },
      },
    },
  };
}
