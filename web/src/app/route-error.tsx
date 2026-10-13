import type { ErrorComponentProps } from "@tanstack/react-router";

import { isApiError } from "../api/errors";

export function RouteError({ error }: ErrorComponentProps) {
  return (
    <p role="alert" className="text-fail-ink">
      {isApiError(error) ? error.title : String(error)}
    </p>
  );
}
