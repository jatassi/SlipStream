import type { ComponentProps } from "react";
import { Empty as KitEmpty } from "../ui/empty";

export const Empty = (props: Omit<ComponentProps<typeof KitEmpty>, "look">) => (
  <KitEmpty look="sheet" {...props} />
);
