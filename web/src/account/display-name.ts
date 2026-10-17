import { z } from "zod";

export const nameField = z
  .string()
  .trim()
  .min(1, "Enter a name")
  .max(100, "Use 100 characters or fewer");
