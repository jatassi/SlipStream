package timingtest

import "math/rand/v2"

func Seeded(seed uint64) *rand.Rand { return rand.New(rand.NewPCG(seed, 0)) }
