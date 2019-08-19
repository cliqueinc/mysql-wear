package util

import (
	"fmt"
	"testing"
)

func TestRandomInt(t *testing.T) {
	// Generate 100k ints, none should dup when starting at 10mil - 1bil
	intMap := map[int]int{}
	dups := 0
	for i := 0; i <= 100000; i++ {
		newRandInt := RandomInt(0, 1000000000)
		if _, inMap := intMap[newRandInt]; inMap {
			fmt.Printf("Warning: Found a duplicated number %d on try %d\n", newRandInt, i)
			dups += 1
		}
		intMap[newRandInt] = 1
	}
	if dups > 50 {
		t.Errorf("Found too many duplicate random ints (%d)", dups)
	}

}

func TestRandomString(t *testing.T) {
	strMap := map[string]int{}
	dups := 0
	for i := 0; i <= 100000; i++ {
		newRandStr := RandomString(15)
		if len(newRandStr) != 15 {
			t.Fatal("RandomString did not return a string of len 15")
		}
		if _, inMap := strMap[newRandStr]; inMap {
			fmt.Printf("Warning: Found a duplicated str using RandomString %s on try %d\n", newRandStr, i)
			dups += 1
		}
		strMap[newRandStr] = 1
	}
	if dups > 50 {
		t.Errorf("Found too many duplicate random strings (%d)", dups)
	}
}
