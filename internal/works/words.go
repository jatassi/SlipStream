package works

import (
	"fmt"
	"strconv"
	"strings"
)

// NumbersWords is ns in words with the noun: "Episode 4", "Episodes 1 and
// 2", "Episodes 1 to 10", "Episodes 1, 2 and 5". Runs of three or more
// fold.
func NumbersWords(noun Noun, ns []int) string {
	var runs []string
	for i := 0; i < len(ns); {
		k := i
		for k+1 < len(ns) && ns[k+1] == ns[k]+1 {
			k++
		}
		if k-i >= 2 {
			runs = append(runs, fmt.Sprintf("%d to %d", ns[i], ns[k]))
		} else {
			for _, n := range ns[i : k+1] {
				runs = append(runs, strconv.Itoa(n))
			}
		}
		i = k + 1
	}
	if len(ns) == 1 {
		return noun.One + " " + runs[0]
	}
	return noun.Many + " " + JoinWords(runs)
}

// JoinWords is a list in a sentence: "a", "a and b", "a, b and c".
func JoinWords(ws []string) string {
	if len(ws) < 2 {
		return strings.Join(ws, "")
	}
	return strings.Join(ws[:len(ws)-1], ", ") + " and " + ws[len(ws)-1]
}

// Places is each of items' place in its level, counted in the order given,
// which is a Tree's: withdrawn Items count where the Tree puts them.
func Places(items []Item) map[ItemID]int {
	place := map[LevelID]int{}
	out := make(map[ItemID]int, len(items))
	for _, it := range items {
		place[it.LevelID]++
		out[it.ID] = place[it.LevelID]
	}
	return out
}
