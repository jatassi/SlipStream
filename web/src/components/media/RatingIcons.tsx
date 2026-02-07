import rtFreshImg from '@/assets/ratings/rt-fresh.png'
import rtRottenImg from '@/assets/ratings/rt-rotten.png'
import imdbImg from '@/assets/ratings/imdb.png'
import metacriticImg from '@/assets/ratings/metacritic.png'

interface IconProps {
  className?: string
}

export function RTFreshIcon({ className = 'h-5' }: IconProps) {
  return <img src={rtFreshImg} alt="Fresh" className={className} />
}

export function RTRottenIcon({ className = 'h-5' }: IconProps) {
  return <img src={rtRottenImg} alt="Rotten" className={className} />
}

export function IMDbIcon({ className = 'h-4' }: IconProps) {
  return <img src={imdbImg} alt="IMDb" className={className} />
}

export function MetacriticIcon({ className = 'h-5' }: IconProps) {
  return <img src={metacriticImg} alt="Metacritic" className={className} />
}
