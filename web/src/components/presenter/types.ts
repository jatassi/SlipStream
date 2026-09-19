export type ActionItem = {
  label: string
  onClick?: () => void
  destructive?: boolean
  confirm?: ActionConfirm
}

export type ActionConfirm = {
  title: string
  description: string
  actions: ActionItem[]
}

export function safeActions(actions: ActionItem[]): ActionItem[] {
  return actions.filter((action) => action.destructive !== true)
}

export function destructiveActions(actions: ActionItem[]): ActionItem[] {
  return actions.filter((action) => action.destructive === true)
}
