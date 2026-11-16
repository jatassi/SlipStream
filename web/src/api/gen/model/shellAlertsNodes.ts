/**
 * warn while a Node has been offline longer than a brief drop, or the Controller is doing a Role's work for an offline Node; fail while a Node has been offline or on another version longer than the offline grace; null otherwise
 * @nullable
 */
export type ShellAlertsNodes = "warn" | "fail" | null;
