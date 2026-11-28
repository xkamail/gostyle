package kindsuffix

type index map[string]int

type T struct {
	NameStr string // want `NameStr: drop the "Str" suffix; the type \(string\) already says it`
	index
}

func f(usersMap map[string]int, idx index) { // want `usersMap: drop the "Map" suffix; the type \(map\[string\]int\) already says it`
	var userSlice []string      // want `userSlice: drop the "Slice" suffix; the type \(\[\]string\) already says it`
	countInt := 0               // want `countInt: drop the "Int" suffix; the type \(int\) already says it`
	doneChan := make(chan bool) // want `doneChan: drop the "Chan" suffix; the type \(chan bool\) already says it`
	lookupMap := idx            // want `lookupMap: drop the "Map" suffix; the type \(kindsuffix.index\) already says it`
	var names []string
	var point, str, maxPrint int
	var nameMap string
	_, _, _, _, _, _, _, _, _, _ = usersMap, userSlice, countInt, doneChan, lookupMap, names, point, str, maxPrint, nameMap
}

func g(tagStr string) {
	tag := []byte(tagStr)
	_ = tag
}
