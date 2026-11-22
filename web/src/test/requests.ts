import { worker } from "./worker";

/** Counts each request by method and path, with a Node's id read as :id. */
export function countRequests() {
  const counts = new Map<string, number>();
  const count = ({ request }: { request: Request }) => {
    const path = new URL(request.url).pathname.replace(
      /^\/api\/v1\/nodes\/[^/]+\//u,
      "/api/v1/nodes/:id/",
    );
    const key = `${request.method} ${path}`;
    counts.set(key, (counts.get(key) ?? 0) + 1);
  };
  worker.events.on("request:start", count);
  return (key: string) => counts.get(key) ?? 0;
}
