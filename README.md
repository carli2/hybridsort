# hybridsort

A generic hybrid sorting algorithm for Go that exploits natural order in data.

```go
import "github.com/carli2/hybridsort"

// Value-based: full hybrid sort with run detection + buffered merge
data := []int{5, 3, 1, 4, 2}
hybridsort.HybridSort(data, func(a, b int) bool { return a < b })

// Index-based: drop-in replacement for sort.Slice
hybridsort.Slice(data, func(i, j int) bool { return data[i] < data[j] })
```

## API

- **`HybridSort[T any](data []T, less func(a, b T) bool)`** — full hybrid sort with natural run detection and buffered merge. Fastest on partially sorted data.
- **`Slice[T any](data []T, less func(i, j int) bool)`** — drop-in replacement for `sort.Slice`. Uses quicksort with insertion sort for small partitions. Zero allocations.
- **`QuickSort[T any](data []T, less func(a, b T) bool)`** — standalone generic quicksort with median-of-3 pivot and insertion sort fallback.

## How it works

HybridSort scans the input for **natural runs** — ascending, descending, and unordered regions — then combines them using a buffered merge. This makes it extremely fast on partially sorted data while remaining competitive on random input.

1. **Scan** the slice into natural blocks (ascending / descending / unordered)
2. **Normalize** blocks: reverse descending runs, quicksort unordered regions
3. **Merge** blocks pairwise using a buffered O(n) merge with an n/2 auxiliary buffer
4. **Fast path** for n ≤ 16: direct insertion sort, zero heap allocations

## Benchmarks

All benchmarks measured on AMD Ryzen 9 7900X3D, Go 1.22, linux/amd64.

### HybridSort vs sort.Sort — tiny inputs (n = 1–10)

Insertion sort fast path, zero heap allocations.

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

### HybridSort vs sort.Sort — random data

| Size | HybridSort | QuickSort | sort.Sort |
|------|----------:|----------:|----------:|
| 100 | 2,625 ns | 1,120 ns | 1,768 ns |
| 1K | 29,641 ns | 17,352 ns | 26,076 ns |
| 10K | 628 µs | 464 µs | 660 µs |
| 100K | 6.9 ms | 6.0 ms | 8.4 ms |

On fully random data, HybridSort is faster than `sort.Sort` at all sizes. The standalone `QuickSort` is fastest here since it has zero overhead from run detection.

### HybridSort vs sort.Sort — sorted data (best case)

Data is already in ascending order — HybridSort detects a single run and returns.

| Size | HybridSort | sort.Sort | Speedup |
|------|----------:|----------:|--------:|
| 100 | 216 ns | 216 ns | 1.0x |
| 1K | 1,890 ns | 1,746 ns | 0.9x |
| 10K | 16.7 µs | 17.2 µs | **1.03x** |
| 100K | 169 µs | 188 µs | **1.11x** |

Both algorithms handle sorted data in O(n). Performance is nearly identical, with HybridSort pulling slightly ahead at larger sizes.

### HybridSort vs sort.Sort — reversed data

Data is in descending order — HybridSort detects one descending run and reverses it.

| Size | HybridSort | sort.Sort | Speedup |
|------|----------:|----------:|--------:|
| 100 | 263 ns | 287 ns | **1.09x** |
| 1K | 2,206 ns | 2,411 ns | **1.09x** |
| 10K | 19.2 µs | 24.7 µs | **1.29x** |
| 100K | 213 µs | 256 µs | **1.20x** |

### HybridSort vs sort.Sort — two presorted blocks (90% + 10%)

Two ascending runs whose values interleave, requiring a real merge.
This is where HybridSort's run detection + buffered merge shines.

| Size | HybridSort | sort.Sort | Speedup |
|------|----------:|----------:|--------:|
| 100 | 448 ns | 2,505 ns | **5.6x** |
| 1K | 3,993 ns | 48,647 ns | **12x** |
| 10K | 38.8 µs | 744 µs | **19x** |
| 100K | 399 µs | 8.8 ms | **22x** |

### HybridSort vs sort.Sort — sorted + random tail (90%/10%)

Common real-world pattern: mostly sorted data with fresh unsorted elements appended.

| Size | HybridSort | sort.Sort | Speedup |
|------|----------:|----------:|--------:|
| 100 | 503 ns | 2,209 ns | **4.4x** |
| 1K | 5,084 ns | 40,158 ns | **7.9x** |
| 10K | 59.1 µs | 571 µs | **9.7x** |
| 100K | 1.58 ms | 7.16 ms | **4.5x** |

### Slice vs sort.Slice — random data

`Slice` is a drop-in replacement for `sort.Slice` with the same `less(i, j int)` signature.

| Size | Slice | sort.Slice | Speedup |
|------|------:|-----------:|--------:|
| 100 | 1,400 ns | 1,499 ns | **1.07x** |
| 1K | 20,220 ns | 22,320 ns | **1.10x** |
| 10K | 560 µs | 597 µs | **1.07x** |
| 100K | 7.35 ms | 7.73 ms | **1.05x** |

### Slice vs sort.Slice — tiny inputs (n = 1–10)

| n | Slice | sort.Slice | Speedup |
|---|------:|-----------:|--------:|
| 1 | 2.8 ns | 35 ns | **12x** |
| 2 | 5.4 ns | 60 ns | **11x** |
| 3 | 9.9 ns | 70 ns | **7x** |
| 4 | 14 ns | 73 ns | **5x** |
| 5 | 16 ns | 76 ns | **5x** |
| 6 | 23 ns | 88 ns | **3.8x** |
| 7 | 24 ns | 88 ns | **3.7x** |
| 8 | 40 ns | 118 ns | **3.0x** |
| 9 | 38 ns | 112 ns | **2.9x** |
| 10 | 72 ns | 162 ns | **2.3x** |

## Design trade-offs

- **Memory**: HybridSort allocates an n/2 buffer for the merge phase. For n ≤ 16 and for Slice, zero allocations. `sort.Sort` uses O(1) extra memory but pays for it with slower merging.
- **Stability**: HybridSort is **not** stable (quicksort is used for unordered blocks).
- **Generics**: Uses Go generics — no interface boxing overhead, unlike `sort.Sort`/`sort.Slice` which require indirection through `sort.Interface` or `reflect.Swapper`.
- **Slice vs HybridSort**: `Slice` uses pure quicksort because the buffered merge requires value-based comparison, which is incompatible with the index-based `less(i, j int)` signature. For maximum performance on partially sorted data, prefer `HybridSort`.

## Install

```
go get github.com/carli2/hybridsort
```
