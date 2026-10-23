import type { ShellAlertsHealth } from "./shellAlertsHealth.ts";
import type { ShellAlertsNodes } from "./shellAlertsNodes.ts";
import type { ShellAlertsTasks } from "./shellAlertsTasks.ts";

export interface ShellAlerts {
  /**
   * The worst level across every Health category, null while nothing is wrong
   * @nullable
   */
  health: ShellAlertsHealth;
  /**
   * warn while a Node is away past the blip or the Controller covers a Role for one, fail while a Node is out of step past the offline grace, null otherwise
   * @nullable
   */
  nodes: ShellAlertsNodes;
  /**
   * The worst level in the Scheduled tasks Health category, null while nothing is wrong
   * @nullable
   */
  tasks: ShellAlertsTasks;
}
