package types

import "fmt"

func ExampleParseTags() {
	fmt.Println(WrapStringToStringMap(ParseTags(`puppet:"hey\"ho" lyra:"here we go"`)))

	// Output: {'lyra' => 'here we go', 'puppet' => 'hey"ho'}
}
