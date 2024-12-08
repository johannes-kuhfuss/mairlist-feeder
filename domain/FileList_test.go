// package domain defines the core data structures
package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsPathNoMatch(t *testing.T) {
	fi1 := FileInfo{Path: "A"}
	fi2 := FileInfo{Path: "B"}
	fl := FileList{fi1, fi2}
	contains := fl.ContainsPath("C")
	assert.False(t, contains)
}

func TestContainsPathMatch(t *testing.T) {
	fi1 := FileInfo{Path: "A"}
	fi2 := FileInfo{Path: "B"}
	fl := FileList{fi1, fi2}
	contains := fl.ContainsPath("B")
	assert.True(t, contains)
}
