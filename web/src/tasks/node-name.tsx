import { Link } from "@tanstack/react-router";

/** The Node a unit runs on, linked to it, or marked removed once it is gone. */
export function NodeName({
  name,
  nodeId,
  removed,
}: {
  name: string;
  nodeId: string;
  removed: boolean;
}) {
  if (name === "") {
    return <>No longer paired</>;
  }
  if (removed) {
    return <>{name} (removed)</>;
  }
  return (
    <Link
      to="/nodes/$nodeId"
      params={{ nodeId }}
      className="rounded-item underline-offset-4 hover:underline"
    >
      {name}
    </Link>
  );
}
