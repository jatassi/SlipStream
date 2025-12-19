import { useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { Search, Plus, Bell } from 'lucide-react'
import { Input } from '@/components/ui/input'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useUIStore } from '@/stores'
import { Badge } from '@/components/ui/badge'

export function Header() {
  const navigate = useNavigate()
  const [searchQuery, setSearchQuery] = useState('')
  const { notifications, dismissNotification } = useUIStore()

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    if (searchQuery.trim()) {
      // Navigate to search or filter current view
      console.log('Search:', searchQuery)
    }
  }

  return (
    <header className="flex h-14 items-center gap-4 border-b border-border bg-card px-6">
      {/* Search */}
      <form onSubmit={handleSearch} className="flex-1 max-w-md">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            type="search"
            placeholder="Search movies, series..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>
      </form>

      {/* Actions */}
      <div className="ml-auto flex items-center gap-2">
        {/* Add dropdown */}
        <DropdownMenu>
          <DropdownMenuTrigger
            className="inline-flex items-center justify-center gap-2 whitespace-nowrap font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0 bg-primary text-primary-foreground shadow hover:bg-primary/90 rounded-md px-3 text-sm h-8"
          >
            <Plus className="size-4" />
            Add
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => navigate({ to: '/movies/add' })}>
              Add Movie
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => navigate({ to: '/series/add' })}>
              Add Series
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        {/* Notifications */}
        <DropdownMenu>
          <DropdownMenuTrigger className="relative inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring hover:bg-accent hover:text-accent-foreground h-9 w-9">
            <Bell className="size-5" />
            {notifications.length > 0 && (
              <Badge
                variant="destructive"
                className="absolute -right-1 -top-1 size-5 p-0 text-xs flex items-center justify-center"
              >
                {notifications.length}
              </Badge>
            )}
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-80">
            {notifications.length === 0 ? (
              <div className="p-4 text-center text-sm text-muted-foreground">
                No notifications
              </div>
            ) : (
              notifications.slice(0, 5).map((notification) => (
                <DropdownMenuItem
                  key={notification.id}
                  onClick={() => dismissNotification(notification.id)}
                  className="flex flex-col items-start gap-1 p-3"
                >
                  <span className="font-medium">{notification.title}</span>
                  {notification.message && (
                    <span className="text-sm text-muted-foreground">
                      {notification.message}
                    </span>
                  )}
                </DropdownMenuItem>
              ))
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  )
}
