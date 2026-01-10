import { Link } from '@tanstack/react-router'
import { MoreHorizontal, Search, Trash2, RefreshCw, Eye } from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { StatusBadge } from '@/components/media/StatusBadge'
import { MovieAvailabilityBadge } from '@/components/media/AvailabilityBadge'
import { formatBytes, formatDate } from '@/lib/formatters'
import type { Movie } from '@/types'

interface MovieTableProps {
  movies: Movie[]
  onSearch?: (id: number) => void
  onRefresh?: (id: number) => void
  onDelete?: (id: number) => void
}

export function MovieTable({
  movies,
  onSearch,
  onRefresh,
  onDelete,
}: MovieTableProps) {
  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Title</TableHead>
            <TableHead>Year</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Release</TableHead>
            <TableHead>Size</TableHead>
            <TableHead>Added</TableHead>
            <TableHead className="w-[70px]"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {movies.map((movie) => (
            <TableRow key={movie.id}>
              <TableCell>
                <Link
                  to="/movies/$id"
                  params={{ id: String(movie.id) }}
                  className="font-medium hover:underline"
                >
                  {movie.title}
                </Link>
              </TableCell>
              <TableCell>{movie.year || '-'}</TableCell>
              <TableCell>
                <StatusBadge status={movie.status} />
              </TableCell>
              <TableCell>
                <MovieAvailabilityBadge movie={movie} />
              </TableCell>
              <TableCell>
                {movie.sizeOnDisk ? formatBytes(movie.sizeOnDisk) : '-'}
              </TableCell>
              <TableCell>{formatDate(movie.addedAt)}</TableCell>
              <TableCell>
                <DropdownMenu>
                  <DropdownMenuTrigger className="inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring hover:bg-accent hover:text-accent-foreground size-8">
                    <MoreHorizontal className="size-4" />
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <Link to="/movies/$id" params={{ id: String(movie.id) }}>
                      <DropdownMenuItem>
                        <Eye className="size-4 mr-2" />
                        View details
                      </DropdownMenuItem>
                    </Link>
                    {onSearch && (
                      <DropdownMenuItem onClick={() => onSearch(movie.id)}>
                        <Search className="size-4 mr-2" />
                        Search
                      </DropdownMenuItem>
                    )}
                    {onRefresh && (
                      <DropdownMenuItem onClick={() => onRefresh(movie.id)}>
                        <RefreshCw className="size-4 mr-2" />
                        Refresh
                      </DropdownMenuItem>
                    )}
                    {onDelete && (
                      <DropdownMenuItem
                        onClick={() => onDelete(movie.id)}
                        className="text-destructive"
                      >
                        <Trash2 className="size-4 mr-2" />
                        Delete
                      </DropdownMenuItem>
                    )}
                  </DropdownMenuContent>
                </DropdownMenu>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
