import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from '@/components/ui/pagination'

import { getPaginationPages } from './history-utils'

type HistoryPaginationProps = {
  page: number
  totalPages: number
  onPreviousPage: () => void
  onNextPage: () => void
  onPageSelect: (page: number) => void
}

export function HistoryPagination({
  page,
  totalPages,
  onPreviousPage,
  onNextPage,
  onPageSelect,
}: HistoryPaginationProps) {
  if (totalPages <= 1) {
    return null
  }

  const pages = getPaginationPages(page, totalPages)

  return (
    <Pagination className="mt-4">
      <PaginationContent>
        <PaginationItem>
          <PaginationPrevious
            onClick={onPreviousPage}
            className={page === 1 ? 'pointer-events-none opacity-50' : 'cursor-pointer'}
          />
        </PaginationItem>
        {pages.map((p) =>
          typeof p === 'string' ? (
            <PaginationItem key={p}>
              <PaginationEllipsis />
            </PaginationItem>
          ) : (
            <PaginationItem key={p}>
              <PaginationLink
                isActive={p === page}
                onClick={() => onPageSelect(p)}
                className="cursor-pointer"
              >
                {p}
              </PaginationLink>
            </PaginationItem>
          ),
        )}
        <PaginationItem>
          <PaginationNext
            onClick={onNextPage}
            className={
              page === totalPages ? 'pointer-events-none opacity-50' : 'cursor-pointer'
            }
          />
        </PaginationItem>
      </PaginationContent>
    </Pagination>
  )
}
