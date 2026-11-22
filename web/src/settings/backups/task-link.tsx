import { Link } from "@tanstack/react-router";

export function TaskLink() {
  return (
    <Link
      to="/tasks/$taskId"
      params={{ taskId: "backup" }}
      className="underline decoration-ink-3 underline-offset-4 hover:decoration-ink-1"
    >
      Open the Backup task
    </Link>
  );
}
