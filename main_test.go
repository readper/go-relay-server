package main

import (
	"testing"
)

func TestRandStr(t *testing.T) {
	str := randStr(1000)
	if len(str) != 1000 {
		t.Errorf("str len %d != %d", len(str), 1000)
	}
}
