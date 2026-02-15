import { User } from 'lucide-react'

import type { Person } from '@/types'

export function CastList({ cast }: { cast: Person[] }) {
  return (
    <div>
      <h3 className="mb-2 text-sm font-semibold">Cast</h3>
      <div className="flex gap-4 overflow-x-auto pb-2">
        {cast.slice(0, 12).map((person) => (
          <CastMember key={person.id} person={person} />
        ))}
      </div>
    </div>
  )
}

function CastMember({ person }: { person: Person }) {
  return (
    <div className="flex w-20 shrink-0 flex-col items-center gap-1">
      <div className="bg-muted flex size-16 items-center justify-center overflow-hidden rounded-full">
        {person.photoUrl ? (
          <img src={person.photoUrl} alt={person.name} className="size-full object-cover" />
        ) : (
          <User className="text-muted-foreground size-8" />
        )}
      </div>
      <span className="line-clamp-2 w-full text-center text-xs">{person.name}</span>
      {person.role ? (
        <span className="text-muted-foreground line-clamp-2 w-full text-center text-xs">
          {person.role}
        </span>
      ) : null}
    </div>
  )
}
