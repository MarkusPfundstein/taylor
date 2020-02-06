package util

func IsSubsetString(set1 []string, set2 []string) bool {
	nope := false
	for _, requirement := range set1 {
		isInIt := false
		for _, capability := range set2 {
			if requirement == capability {
				isInIt = true
				break
			}
		}
		if isInIt == false {
			nope = true
		}
	}

	// Attention: inverse
	return !nope
}
