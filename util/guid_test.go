package util

import (
	"log"
	"testing"
)

func TestUtilsGuid(t *testing.T) {
	/*
	   if testing.Short() {
	       t.Skip("skipping test in short mode.")
	   }
	   func BenchmarkXxx(*testing.B)
	   go test -bench
	*/

	var mySet = map[string]string{}
	for i := 0; i < 50000; i++ {

		// Create a bucket free ID
		b := NewGuid()

		if _, ok := mySet[b]; ok {
			log.Println("Duped id!" + b)
			t.Fail()
		}
		mySet[b] = "in"

		c := NewGuidFromShard(b)
		if _, ok := mySet[c]; ok {
			log.Println("Duped id!" + c)
			t.Fail()
		}
		mySet[c] = "in"

		bid, errb := GetShardId(b)
		cid, errc := GetShardId(c)

		if errb != nil {
			t.Fail()
		}
		if errc != nil {
			t.Fail()
		}

		if bid != cid {
			t.Fail()
		}

	}
	log.Println("Created 50,000 Guids, no dups")
}

func TestGuids(t *testing.T) {
	log.Println(NewGuidFromShard("kjdzhg6nzai6jgkkgq24fsahjq"))
	log.Println(NewGuidFromShard("kjdzhg6nzai6jgkkgq24fsahjq"))
	log.Println(NewGuidFromShard("kjdzhg6nzai6jgkkgq24fsahjq"))
	log.Println(NewGuidFromShard("kjdzhg6nzai6jgkkgq24fsahjq"))
	log.Println(NewGuidFromShard("kjdzhg6nzai6jgkkgq24fsahjq"))
	log.Println(NewGuidFromShard("kjdzhg6nzai6jgkkgq24fsahjq"))
	log.Println(NewGuidFromShard("ijo6p5phuui6jhhfgq24fsahjq"))
	log.Println(NewGuidFromShard("ijo6p5phuui6jhhfgq24fsahjq"))
}
