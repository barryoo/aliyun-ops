package utils

func IfTrue(condition bool, a, b interface{}) interface{} {
	if condition {
		return a
	}
	return b
}
