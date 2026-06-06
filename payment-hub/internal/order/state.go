package order

var transitions = map[string]map[string]bool{
	"pending": {
		"processing": true,
		"expired":    true,
	},
	"processing": {
		"success": true,
		"failed":  true,
	},
	"success": {
		"refunded": true,
	},
}

func CanTransition(from, to string) bool {
	if from == to {
		return true
	}
	next, ok := transitions[from]
	if !ok {
		return false
	}
	return next[to]
}

func IsFinalStatus(status string) bool {
	switch status {
	case "success", "failed", "expired", "refunded":
		return true
	default:
		return false
	}
}
