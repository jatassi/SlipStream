import type { ShellAlertsNodes } from "./shellAlertsNodes.ts";

export interface ShellAlerts {
  /**
   * warn while a Node is away past the blip or the Controller covers a Role for one, fail while a Node is out of step past the offline grace, null otherwise
   * @nullable
   */
  nodes: ShellAlertsNodes;
}
