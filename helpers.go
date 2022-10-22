package asyncp

func mergeStrArr(a ...[]string) []string {
	if len(a) == 0 {
		return nil
	}
	mp := map[string]bool{}
	res := make([]string, 0, len(a[0])*len(a))
	for _, v := range a {
		for _, s := range v {
			if !mp[s] {
				mp[s] = true
				res = append(res, s)
			}
		}
	}
	return res
}

func excludeFromStrArr(a []string, excl ...string) []string {
	if len(excl) == 0 {
		return append(make([]string, 0, len(a)), a...)
	}
	res := make([]string, 0, len(a))
mainLoop:
	for _, v := range a {
		for _, s := range excl {
			if s == v {
				continue mainLoop
			}
		}
		res = append(res, v)
	}
	return res
}
