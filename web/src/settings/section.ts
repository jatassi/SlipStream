import type { QueryClient } from "@tanstack/react-query";
import type { ComponentType } from "react";

export type SettingsSection = {
  readonly id: string;
  readonly label: string;
  readonly load: (queryClient: QueryClient) => Promise<void>;
  readonly component: ComponentType;
};
