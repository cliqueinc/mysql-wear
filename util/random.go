package util

import (
	"math/rand"
	"time"
)

var ranStrSetAlphaNum = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func init() {
	// Seed the random num gen, once for app
	rand.Seed(time.Now().UTC().UnixNano())
}

func RandomInt(min, max int) int {
	/* We will give back min - max inclusive.
	   0,0 always gives back 0
	   1,1 always gives back 1
	   1,2 gives back either 1 or 2
	   etc
	   1,0 panics (Intn panics on negative #)
	*/
	return min + rand.Intn(max+1-min)
}

func RandomString(length uint) string {
	if length < 1 {
		msg := "Cant ask for random string of length less than 1"
		panic(msg)
	}
	bytes := make([]byte, length)
	for i := range bytes {
		bytes[i] = ranStrSetAlphaNum[rand.Intn(len(ranStrSetAlphaNum))]
	}
	return string(bytes)
}
