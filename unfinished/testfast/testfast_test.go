package main

import (
	"fmt"
	"testing"
)

func TestWriteFile(t *testing.T) {
	if testing.Short() {
		t.Skipf("SKIP %s: touches filesystem", t.Name())
	}

	// ... test code goes here ...//
}

// TestMul runs serially: no other tests in this package will run while it is running
func TestMul(t *testing.T) {
	if 5*2 != 10 {
		t.Fatal("5*2 != 10")
	}
}

// TestAdd runs in parallel with TestSub, and it's subtests will run in parallel both with each other and TestSub.
func TestAdd(t *testing.T) {
	t.Parallel() // Add will run in parallel with other tests in this package from this point on

	for _, tt := range []struct {
		a, b, want int
	}{
		{2, 2, 4},
		{3, 3, 6},
		{-128, 128, 0},
	} {
		tt := tt // capture range variable: see https://github.com/golang/go/discussions/56010 for details
		t.Run(fmt.Sprintf("%d+%d=%d", tt.a, tt.b, tt.want), func(t *testing.T) {
			t.Parallel() // this subtest will run in parallel with other subtests of TestAdd
			got := tt.a + tt.b
			if got != tt.want {
				t.Errorf("Add(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestSub will run in parallel with other tests in this package,
// but only one of its subtests will run at a time
func TestSub(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		a, b, want int
	}{
		{2, 2, 0},
		{2, -2, 4},
	} {
		t.Run(fmt.Sprintf("%d-%d=%d", tt.a, tt.b, tt.want), func(t *testing.T) {
			got := tt.a - tt.b
			if got != tt.want {
				t.Errorf("Add(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
