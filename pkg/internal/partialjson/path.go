package partialjson

// JoinPath joins a JSON path slice into a dot-separated string.
// Handles array indices like "[0]" correctly (no dot before brackets).
func JoinPath(path []string) string {
	if len(path) == 0 {
		return ""
	}
	result := path[0]
	for i := 1; i < len(path); i++ {
		if len(path[i]) > 0 && path[i][0] == '[' {
			result += path[i] // Array index like "[0]"
		} else {
			result += "." + path[i]
		}
	}
	return result
}

// IsPathOrParentIncomplete checks if a path or any of its parents are in the incomplete set.
// For example, if "user" is incomplete, "user.name" is also considered incomplete.
func IsPathOrParentIncomplete(jsonPath string, incompleteSet map[string]bool) bool {
	if incompleteSet[jsonPath] {
		return true
	}
	// Check parent paths
	for i := len(jsonPath) - 1; i >= 0; i-- {
		if jsonPath[i] == '.' || jsonPath[i] == '[' {
			if incompleteSet[jsonPath[:i]] {
				return true
			}
		}
	}
	return false
}

// BuildIncompleteSet creates a set of incomplete JSON paths for fast lookup.
func BuildIncompleteSet(incompletePaths [][]string) map[string]bool {
	set := make(map[string]bool, len(incompletePaths))
	for _, path := range incompletePaths {
		set[JoinPath(path)] = true
	}
	return set
}
