package hybridsort

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

func intLess(a, b int) bool { return a < b }

// --- Correctness tests ---

func TestHybridSort_Empty(t *testing.T) {
	var data []int
	HybridSort(data, intLess)
}

func TestHybridSort_Single(t *testing.T) {
	data := []int{42}
	HybridSort(data, intLess)
	if data[0] != 42 {
		t.Fatal("single element changed")
	}
}

func TestHybridSort_Sorted(t *testing.T) {
	data := make([]int, 1000)
	for i := range data {
		data[i] = i
	}
	HybridSort(data, intLess)
	assertSorted(t, data)
}

func TestHybridSort_Reversed(t *testing.T) {
	data := make([]int, 1000)
	for i := range data {
		data[i] = len(data) - i
	}
	HybridSort(data, intLess)
	assertSorted(t, data)
}

func TestHybridSort_Random(t *testing.T) {
	rng := rand.New(rand.NewSource(12345))
	data := make([]int, 10000)
	for i := range data {
		data[i] = rng.Intn(100000)
	}
	HybridSort(data, intLess)
	assertSorted(t, data)
}

func TestHybridSort_Duplicates(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	data := make([]int, 5000)
	for i := range data {
		data[i] = rng.Intn(10)
	}
	HybridSort(data, intLess)
	assertSorted(t, data)
}

func TestHybridSort_AllEqual(t *testing.T) {
	data := make([]int, 500)
	for i := range data {
		data[i] = 7
	}
	HybridSort(data, intLess)
	assertSorted(t, data)
}

func TestHybridSort_MixedRuns(t *testing.T) {
	// Create data with alternating ascending and descending runs.
	var data []int
	for i := 0; i < 100; i++ {
		data = append(data, i)
	}
	for i := 200; i > 100; i-- {
		data = append(data, i)
	}
	for i := 201; i < 300; i++ {
		data = append(data, i)
	}
	HybridSort(data, intLess)
	assertSorted(t, data)
}

func TestQuickSort_Random(t *testing.T) {
	rng := rand.New(rand.NewSource(777))
	data := make([]int, 10000)
	for i := range data {
		data[i] = rng.Intn(100000)
	}
	QuickSort(data, intLess)
	assertSorted(t, data)
}

func TestSlice_Random(t *testing.T) {
	rng := rand.New(rand.NewSource(555))
	data := make([]int, 10000)
	for i := range data {
		data[i] = rng.Intn(100000)
	}
	Slice(data, func(i, j int) bool { return data[i] < data[j] })
	assertSorted(t, data)
}

func TestSlice_Small(t *testing.T) {
	for n := 0; n <= 20; n++ {
		data := make([]int, n)
		for i := range data {
			data[i] = n - i
		}
		Slice(data, func(i, j int) bool { return data[i] < data[j] })
		assertSorted(t, data)
	}
}

func assertSorted(t *testing.T, data []int) {
	t.Helper()
	for i := 1; i < len(data); i++ {
		if data[i] < data[i-1] {
			t.Fatalf("not sorted at index %d: %d > %d", i-1, data[i-1], data[i])
		}
	}
}

// --- sort.Interface adapter for stdlib benchmark ---

type intSlice []int

func (s intSlice) Len() int           { return len(s) }
func (s intSlice) Less(i, j int) bool { return s[i] < s[j] }
func (s intSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// --- Benchmarks ---

var sizes = []struct {
	name string
	n    int
}{
	{"100", 100},
	{"1K", 1_000},
	{"10K", 10_000},
	{"100K", 100_000},
}

func makeRandom(n int, seed int64) []int {
	rng := rand.New(rand.NewSource(seed))
	data := make([]int, n)
	for i := range data {
		data[i] = rng.Intn(n * 10)
	}
	return data
}

func makeSorted(n int) []int {
	data := make([]int, n)
	for i := range data {
		data[i] = i
	}
	return data
}

// makeTwoBlocks creates two ascending blocks where values interleave.
// 90% first block, 10% second block. The second block's values fall
// within the first block's range, forcing a real merge.
// n=10: [1,2,3,4,6,7,8,9, 0,5]
func makeTwoBlocks(n int) []int {
	if n < 2 {
		return makeSorted(n)
	}
	data := make([]int, n)
	split := n * 9 / 10
	if split < 1 {
		split = 1
	}
	// Distribute 0..n-1 so that the second block interleaves with the first.
	// Pick every 10th value for the second block, rest goes to the first.
	var first, second []int
	for v := 0; v < n; v++ {
		if v%(n/max(n-split, 1)) == 0 && len(second) < n-split {
			second = append(second, v)
		} else {
			first = append(first, v)
		}
	}
	copy(data, first)
	copy(data[len(first):], second)
	return data
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// makeSortedWithTail creates 90% sorted ascending, then 10% random values appended.
func makeSortedWithTail(n int, seed int64) []int {
	if n < 2 {
		return makeSorted(n)
	}
	data := make([]int, n)
	split := n * 9 / 10
	if split < 1 {
		split = 1
	}
	for i := 0; i < split; i++ {
		data[i] = i
	}
	rng := rand.New(rand.NewSource(seed))
	for i := split; i < n; i++ {
		data[i] = rng.Intn(n)
	}
	return data
}

func makeReversed(n int) []int {
	data := make([]int, n)
	for i := range data {
		data[i] = n - i
	}
	return data
}

// Random data benchmarks

func BenchmarkHybridSort_Random(b *testing.B) {
	for _, sz := range sizes {
		src := makeRandom(sz.n, 42)
		b.Run(sz.name, func(b *testing.B) {
			buf := make([]int, sz.n)
			for i := 0; i < b.N; i++ {
				copy(buf, src)
				HybridSort(buf, intLess)
			}
		})
	}
}

func BenchmarkQuickSort_Random(b *testing.B) {
	for _, sz := range sizes {
		src := makeRandom(sz.n, 42)
		b.Run(sz.name, func(b *testing.B) {
			buf := make([]int, sz.n)
			for i := 0; i < b.N; i++ {
				copy(buf, src)
				QuickSort(buf, intLess)
			}
		})
	}
}

func BenchmarkStdlibSort_Random(b *testing.B) {
	for _, sz := range sizes {
		src := makeRandom(sz.n, 42)
		b.Run(sz.name, func(b *testing.B) {
			buf := make([]int, sz.n)
			for i := 0; i < b.N; i++ {
				copy(buf, src)
				sort.Sort(intSlice(buf))
			}
		})
	}
}

// Sorted data benchmarks

func BenchmarkHybridSort_Sorted(b *testing.B) {
	for _, sz := range sizes {
		src := makeSorted(sz.n)
		b.Run(sz.name, func(b *testing.B) {
			buf := make([]int, sz.n)
			for i := 0; i < b.N; i++ {
				copy(buf, src)
				HybridSort(buf, intLess)
			}
		})
	}
}

func BenchmarkStdlibSort_Sorted(b *testing.B) {
	for _, sz := range sizes {
		src := makeSorted(sz.n)
		b.Run(sz.name, func(b *testing.B) {
			buf := make([]int, sz.n)
			for i := 0; i < b.N; i++ {
				copy(buf, src)
				sort.Sort(intSlice(buf))
			}
		})
	}
}

// Reversed data benchmarks

func BenchmarkHybridSort_Reversed(b *testing.B) {
	for _, sz := range sizes {
		src := makeReversed(sz.n)
		b.Run(sz.name, func(b *testing.B) {
			buf := make([]int, sz.n)
			for i := 0; i < b.N; i++ {
				copy(buf, src)
				HybridSort(buf, intLess)
			}
		})
	}
}

// Two presorted blocks (90%/10%) benchmarks

func BenchmarkHybridSort_TwoBlocks(b *testing.B) {
	for _, sz := range sizes {
		src := makeTwoBlocks(sz.n)
		b.Run(sz.name, func(b *testing.B) {
			buf := make([]int, sz.n)
			for i := 0; i < b.N; i++ {
				copy(buf, src)
				HybridSort(buf, intLess)
			}
		})
	}
}

func BenchmarkStdlibSort_TwoBlocks(b *testing.B) {
	for _, sz := range sizes {
		src := makeTwoBlocks(sz.n)
		b.Run(sz.name, func(b *testing.B) {
			buf := make([]int, sz.n)
			for i := 0; i < b.N; i++ {
				copy(buf, src)
				sort.Sort(intSlice(buf))
			}
		})
	}
}

// 90% sorted + 10% random tail benchmarks

func BenchmarkHybridSort_SortedTail(b *testing.B) {
	for _, sz := range sizes {
		src := makeSortedWithTail(sz.n, 42)
		b.Run(sz.name, func(b *testing.B) {
			buf := make([]int, sz.n)
			for i := 0; i < b.N; i++ {
				copy(buf, src)
				HybridSort(buf, intLess)
			}
		})
	}
}

func BenchmarkStdlibSort_SortedTail(b *testing.B) {
	for _, sz := range sizes {
		src := makeSortedWithTail(sz.n, 42)
		b.Run(sz.name, func(b *testing.B) {
			buf := make([]int, sz.n)
			for i := 0; i < b.N; i++ {
				copy(buf, src)
				sort.Sort(intSlice(buf))
			}
		})
	}
}

func BenchmarkStdlibSort_Reversed(b *testing.B) {
	for _, sz := range sizes {
		src := makeReversed(sz.n)
		b.Run(sz.name, func(b *testing.B) {
			buf := make([]int, sz.n)
			for i := 0; i < b.N; i++ {
				copy(buf, src)
				sort.Sort(intSlice(buf))
			}
		})
	}
}

// Slice vs sort.Slice benchmarks

func BenchmarkSlice_Random(b *testing.B) {
	for _, sz := range sizes {
		src := makeRandom(sz.n, 42)
		b.Run(sz.name, func(b *testing.B) {
			buf := make([]int, sz.n)
			for i := 0; i < b.N; i++ {
				copy(buf, src)
				Slice(buf, func(i, j int) bool { return buf[i] < buf[j] })
			}
		})
	}
}

func BenchmarkStdlibSlice_Random(b *testing.B) {
	for _, sz := range sizes {
		src := makeRandom(sz.n, 42)
		b.Run(sz.name, func(b *testing.B) {
			buf := make([]int, sz.n)
			for i := 0; i < b.N; i++ {
				copy(buf, src)
				sort.Slice(buf, func(i, j int) bool { return buf[i] < buf[j] })
			}
		})
	}
}

func BenchmarkSlice_Tiny(b *testing.B) {
	for n := 1; n <= 10; n++ {
		src := makeRandom(n, int64(n))
		name := fmt.Sprintf("n=%d", n)
		b.Run(name, func(b *testing.B) {
			buf := make([]int, n)
			for i := 0; i < b.N; i++ {
				copy(buf, src)
				Slice(buf, func(i, j int) bool { return buf[i] < buf[j] })
			}
		})
	}
}

func BenchmarkStdlibSlice_Tiny(b *testing.B) {
	for n := 1; n <= 10; n++ {
		src := makeRandom(n, int64(n))
		name := fmt.Sprintf("n=%d", n)
		b.Run(name, func(b *testing.B) {
			buf := make([]int, n)
			for i := 0; i < b.N; i++ {
				copy(buf, src)
				sort.Slice(buf, func(i, j int) bool { return buf[i] < buf[j] })
			}
		})
	}
}

// Per-element benchmarks for n=1..10

func BenchmarkHybridSort_Tiny(b *testing.B) {
	for n := 1; n <= 10; n++ {
		src := makeRandom(n, int64(n))
		name := fmt.Sprintf("n=%d", n)
		b.Run(name, func(b *testing.B) {
			buf := make([]int, n)
			for i := 0; i < b.N; i++ {
				copy(buf, src)
				HybridSort(buf, intLess)
			}
		})
	}
}

func BenchmarkStdlibSort_Tiny(b *testing.B) {
	for n := 1; n <= 10; n++ {
		src := makeRandom(n, int64(n))
		name := fmt.Sprintf("n=%d", n)
		b.Run(name, func(b *testing.B) {
			buf := make([]int, n)
			for i := 0; i < b.N; i++ {
				copy(buf, src)
				sort.Sort(intSlice(buf))
			}
		})
	}
}
