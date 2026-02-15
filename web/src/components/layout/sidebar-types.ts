export type NavItem = {
  title: string
  href: string
  icon: React.ElementType
  theme?: 'movie' | 'tv'
}

export type ActionItem = {
  title: string
  icon: React.ElementType
  action: 'restart' | 'logout'
  variant?: 'warning' | 'destructive'
}

export type MenuItem = NavItem | ActionItem

export function isActionItem(item: MenuItem): item is ActionItem {
  return 'action' in item
}

export type CollapsibleNavGroup = {
  id: string
  title: string
  icon: React.ElementType
  items: MenuItem[]
}
