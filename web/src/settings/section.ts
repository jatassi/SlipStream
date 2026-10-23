import type { QueryClient } from "@tanstack/react-query";
import type { ComponentType } from "react";

import type { HealthCategoryID } from "../api/gen/model";

export type SettingsSection = {
  readonly id: string;
  readonly label: string;
  readonly load: (queryClient: QueryClient) => Promise<void>;
  readonly component: ComponentType;
  /** Drops the 560px column, for an editor that needs the page's width. */
  readonly wide?: boolean;
  /** The Health categories whose problems show on the section's row and at its top. */
  readonly health?: readonly HealthCategoryID[];
};
