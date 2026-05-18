// package domain defines the core data structures
package domain

import (
	"sort"
	"testing"
	"time"

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

func TestSortedReturnsSorted(t *testing.T) {
	fi1 := FileInfo{Path: "A", StartTime: time.Date(2025, 12, 14, 21, 30, 0, 0, time.UTC)}
	fi2 := FileInfo{Path: "B", StartTime: time.Date(2025, 12, 14, 21, 00, 0, 0, time.UTC)}
	fl := FileList{fi1, fi2}
	sort.Sort(fl)
	assert.EqualValues(t, "B", fl[0].Path)
	assert.EqualValues(t, "A", fl[1].Path)
}

func TestParseFolderDateReturnsNormalizedDate(t *testing.T) {
	date, err := ParseFolderDate("2024-09-22")
	assert.Nil(t, err)
	assert.EqualValues(t, time.Date(2024, time.September, 22, 0, 0, 0, 0, time.Local), date)
}

func TestFormatFolderDateReturnsDateString(t *testing.T) {
	date := time.Date(2024, time.September, 22, 13, 15, 0, 0, time.Local)
	assert.EqualValues(t, "2024-09-22", FormatFolderDate(date))
}
