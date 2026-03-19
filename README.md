# hybridsort

A generic hybrid sorting algorithm for Go that exploits natural order in data.

```go
import "github.com/carli2/hybridsort"

data := []int{5, 3, 1, 4, 2}
hybridsort.HybridSort(data, func(a, b int) bool { return a < b })
```

## How it works

HybridSort scans the input for **natural runs** — ascending, descending, and unordered regions — then combines them using a buffered merge. This makes it extremely fast on partially sorted data while remaining competitive on random input.

1. **Scan** the slice into natural blocks (ascending / descending / unordered)
2. **Normalize** blocks: reverse descending runs, quicksort unordered regions
3. **Merge** blocks pairwise using a buffered O(n) merge with an n/2 auxiliary buffer
4. **Fast path** for n ≤ 16: direct insertion sort, zero heap allocations

Also exported: `QuickSort` — a standalone generic quicksort with median-of-3 pivot selection and insertion sort fallback for small partitions.

## Benchmarks

All benchmarks measured on AMD Ryzen 9 7900X3D, Go 1.22, linux/amd64.
Compared against `sort.Sort` from the standard library.

### Tiny inputs (n = 1–10)

HybridSort uses insertion sort for n ≤ 16, avoiding all heap allocations.

| n | HybridSort | sort.Sort | Speedup |
|---|----------:|----------:|--------:|
| 1 | 2.6 ns | 26 ns | **10x** |
| 2 | 5.4 ns | 30 ns | **5.6x** |
| 3 | 9.9 ns | 39 ns | **3.9x** |
| 4 | 14 ns | 44 ns | **3.1x** |
| 5 | 15 ns | 46 ns | **3.1x** |
| 6 | 23 ns | 62 ns | **2.7x** |
| 7 | 23 ns | 61 ns | **2.6x** |
| 8 | 37 ns | 92 ns | **2.5x** |
| 9 | 37 ns | 83 ns | **2.2x** |
| 10 | 66 ns | 140 ns | **2.1x** |

### Random data

| Size | HybridSort | QuickSort | sort.Sort |
|------|----------:|----------:|----------:|
| 100 | 2,625 ns | 1,120 ns | 1,768 ns |
| 1K | 29,641 ns | 17,352 ns | 26,076 ns |
| 10K | 628 µs | 464 µs | 660 µs |
| 100K | 6.9 ms | 6.0 ms | 8.4 ms |

On fully random data, HybridSort is faster than `sort.Sort` at all sizes. The standalone `QuickSort` is fastest here since it has zero overhead from run detection.

### Sorted data (best case)

Data is already in ascending order — HybridSort detects a single run and returns.

| Size | HybridSort | sort.Sort | Speedup |
|------|----------:|----------:|--------:|
| 100 | 216 ns | 216 ns | 1.0x |
| 1K | 1,890 ns | 1,746 ns | 0.9x |
| 10K | 16.7 µs | 17.2 µs | **1.03x** |
| 100K | 169 µs | 188 µs | **1.11x** |

Both algorithms handle sorted data in O(n). Performance is nearly identical, with HybridSort pulling slightly ahead at larger sizes.

### Reversed data

Data is in descending order — HybridSort detects one descending run and reverses it.

| Size | HybridSort | sort.Sort | Speedup |
|------|----------:|----------:|--------:|
| 100 | 263 ns | 287 ns | **1.09x** |
| 1K | 2,206 ns | 2,411 ns | **1.09x** |
| 10K | 19.2 µs | 24.7 µs | **1.29x** |
| 100K | 213 µs | 256 µs | **1.20x** |

### Two presorted blocks (90% + 10%, interleaved values)

Two ascending runs whose values interleave, requiring a real merge.
This is where HybridSort's run detection + buffered merge shines.

| Size | HybridSort | sort.Sort | Speedup |
|------|----------:|----------:|--------:|
| 100 | 448 ns | 2,505 ns | **5.6x** |
| 1K | 3,993 ns | 48,647 ns | **12x** |
| 10K | 38.8 µs | 744 µs | **19x** |
| 100K | 399 µs | 8.8 ms | **22x** |

### Sorted + random tail (90% sorted, 10% random appended)

Common real-world pattern: mostly sorted data with fresh unsorted elements appended.

| Size | HybridSort | sort.Sort | Speedup |
|------|----------:|----------:|--------:|
| 100 | 503 ns | 2,209 ns | **4.4x** |
| 1K | 5,084 ns | 40,158 ns | **7.9x** |
| 10K | 59.1 µs | 571 µs | **9.7x** |
| 100K | 1.58 ms | 7.16 ms | **4.5x** |

## Design trade-offs

- **Memory**: HybridSort allocates an n/2 buffer for the merge phase. For n ≤ 16, zero allocations. `sort.Sort` uses O(1) extra memory but pays for it with slower merging.
- **Stability**: HybridSort is **not** stable (quicksort is used for unordered blocks).
- **Generics**: Uses Go generics (`[T any]` + `less` function) — no interface boxing overhead, unlike `sort.Sort` which requires the `sort.Interface` indirection.

## Install

```
go get github.com/carli2/hybridsort
```
