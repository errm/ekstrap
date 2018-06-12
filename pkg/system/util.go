package system

func set(s map[string]bool, value string) map[string]bool {
	if s == nil {
		s = make(map[string]bool)
	}
	s[value] = true
	return s
}
