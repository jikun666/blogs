package main

type Item struct {
	Val    string
	Weight int

	curWeight int
}

func nextSmoothWeighted(items []*Item) (best int) {
	n := len(items)
	if n == 0 {
		return -1
	}

	best = -1
	total := 0

	for i := 0; i < n; i++ {
		item := items[i]
		total += item.Weight

		item.curWeight += item.Weight
		if best < 0 || item.curWeight > items[best].curWeight {
			best = i
		}
	}

	items[best].curWeight -= total
	return
}

func main() {
	items := []*Item{
		// {"a", 4, 0},
		// {"b", 2, 0},
		// {"c", 1, 0},
		{"a", 6, 0},
		{"b", 4, 0},
	}

	for i := 0; i < 10; i++ {
		best := nextSmoothWeighted(items)
		println(items[best].Val)
	}
}
