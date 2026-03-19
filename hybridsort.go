package hybridsort

type RunType uint8

const (
	Asc RunType = iota
	Desc
	Unordered
)

type block[T any] struct {
	start int
	len   int
	typ   RunType
}

// HybridSort sorts data in ascending order according to less.
//
// Idea:
//   - scan data into natural blocks: asc / desc / unordered
//   - keep at most 4 blocks "active"
//   - small monotone blocks are coalesced into a larger unordered block
//   - unordered blocks are quicksorted, then treated as asc
//   - 4-way merge is performed in-place via pairwise in-place merges
//
// Notes:
//   - "4-way merge in-place" is implemented as:
//       merge(b0,b1), merge(b2,b3), then merge(the two results)
//     which stays in-place, but is not as asymptotically optimal as a
//     dedicated advanced in-place multiway merge.
//   - This implementation is not stable.
func HybridSort[T any](data []T, less func(a, b T) bool) {
	n := len(data)
	if n < 2 {
		return
	}
	// Fast path for small slices: insertion sort, zero allocations.
	if n <= 16 {
		insertionSortRange(data, 0, n-1, less)
		return
	}

	const minMonotone = 8

	buf := make([]T, n/2)
	stack := make([]block[T], 0, 4)

	i := 0
	for i < n {
		b := detectBlock(data, i, less)

		// Merge too-small monotone blocks into a larger unordered block.
		if (b.typ == Asc || b.typ == Desc) && b.len < minMonotone {
			b.typ = Unordered

			// If previous block is adjacent unordered, extend it.
			if len(stack) > 0 {
				top := &stack[len(stack)-1]
				if top.typ == Unordered && top.start+top.len == b.start {
					top.len += b.len
				} else {
					stack = append(stack, b)
				}
			} else {
				stack = append(stack, b)
			}
		} else {
			// If current block is unordered and previous is adjacent unordered, merge them.
			if b.typ == Unordered && len(stack) > 0 {
				top := &stack[len(stack)-1]
				if top.typ == Unordered && top.start+top.len == b.start {
					top.len += b.len
				} else {
					stack = append(stack, b)
				}
			} else {
				stack = append(stack, b)
			}
		}

		i = b.start + b.len

		// Keep only 4 blocks on stack by reducing earliest blocks.
		for len(stack) > 4 {
			reduceStack(data, buf, &stack, less)
		}
	}

	// Normalize all remaining blocks to ascending.
	for idx := range stack {
		normalizeBlock(data, &stack[idx], less)
	}

	// Final 4-way merge (implemented pairwise).
	for len(stack) > 1 {
		reduceSortedStack(data, buf, &stack, less)
	}
}

// detectBlock tries to identify a natural block starting at pos.
// Priority:
//   1) ascending run
//   2) descending run
//   3) unordered region until next clear monotone block
func detectBlock[T any](data []T, pos int, less func(a, b T) bool) block[T] {
	n := len(data)
	if pos >= n-1 {
		return block[T]{start: pos, len: 1, typ: Asc}
	}

	a, b := data[pos], data[pos+1]

	// Ascending: a <= b
	if !less(b, a) {
		j := pos + 2
		for j < n && !less(data[j], data[j-1]) {
			j++
		}
		return block[T]{start: pos, len: j - pos, typ: Asc}
	}

	// Descending: a > b
	if less(b, a) {
		j := pos + 2
		for j < n && less(data[j], data[j-1]) {
			j++
		}
		return block[T]{start: pos, len: j - pos, typ: Desc}
	}

	// Fallback
	return block[T]{start: pos, len: 1, typ: Unordered}
}

// reduceStack reduces stack length when more than 4 blocks are present.
// Strategy:
//   - normalize first two blocks
//   - merge them in-place
//   - keep result as one asc block
func reduceStack[T any](data []T, buf []T, stack *[]block[T], less func(a, b T) bool) {
	s := *stack
	if len(s) < 2 {
		return
	}

	normalizeBlock(data, &s[0], less)
	normalizeBlock(data, &s[1], less)

	bufferedMerge(data, buf, s[0].start, s[0].start+s[0].len, s[0].start+s[0].len+s[1].len, less)

	s[0].len += s[1].len
	s[0].typ = Asc

	copy(s[1:], s[2:])
	s = s[:len(s)-1]
	*stack = s
}

// reduceSortedStack merges already-normalized ascending blocks.
// If 4 blocks exist, it performs a 4-way merge pairwise in-place:
//   merge(b0,b1), merge(b2,b3), merge(result01,result23)
func reduceSortedStack[T any](data []T, buf []T, stack *[]block[T], less func(a, b T) bool) {
	s := *stack

	switch len(s) {
	case 2:
		bufferedMerge(data, buf, s[0].start, s[0].start+s[0].len, s[0].start+s[0].len+s[1].len, less)
		s[0].len += s[1].len
		*stack = s[:1]

	case 3:
		// Merge first two, then with third.
		bufferedMerge(data, buf, s[0].start, s[0].start+s[0].len, s[0].start+s[0].len+s[1].len, less)
		s[0].len += s[1].len

		copy(s[1:], s[2:])
		s = s[:2]

		bufferedMerge(data, buf, s[0].start, s[0].start+s[0].len, s[0].start+s[0].len+s[1].len, less)
		s[0].len += s[1].len
		*stack = s[:1]

	default:
		// len >= 4: pairwise 4-way merge.
		bufferedMerge(data, buf, s[0].start, s[0].start+s[0].len, s[0].start+s[0].len+s[1].len, less)
		leftLen := s[0].len + s[1].len

		bufferedMerge(data, buf, s[2].start, s[2].start+s[2].len, s[2].start+s[2].len+s[3].len, less)
		rightLen := s[2].len + s[3].len

		bufferedMerge(data, buf, s[0].start, s[0].start+leftLen, s[0].start+leftLen+rightLen, less)

		s[0].len = leftLen + rightLen
		s[0].typ = Asc

		if len(s) > 4 {
			copy(s[1:], s[4:])
			s = s[:len(s)-3]
		} else {
			s = s[:1]
		}
		*stack = s
	}
}

func normalizeBlock[T any](data []T, b *block[T], less func(a, b T) bool) {
	switch b.typ {
	case Asc:
		return
	case Desc:
		reverse(data[b.start : b.start+b.len])
		b.typ = Asc
	case Unordered:
		QuickSort(data[b.start:b.start+b.len], less)
		b.typ = Asc
	}
}

func reverse[T any](data []T) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}

// --------------------
// Quicksort (in-place)
// --------------------

func QuickSort[T any](data []T, less func(a, b T) bool) {
	if len(data) < 2 {
		return
	}
	qsort(data, 0, len(data)-1, less)
}

func qsort[T any](data []T, lo, hi int, less func(a, b T) bool) {
	for lo < hi {
		if hi-lo <= 16 {
			insertionSortRange(data, lo, hi, less)
			return
		}

		lt, gt := partition3(data, lo, hi, less)

		// Recurse into smaller side first to bound stack depth.
		if lt-lo < hi-gt {
			qsort(data, lo, lt-1, less)
			lo = gt + 1
		} else {
			qsort(data, gt+1, hi, less)
			hi = lt - 1
		}
	}
}

// partition3 performs 3-way (Dutch National Flag) partitioning.
// Returns lt, gt such that:
//
//	data[lo..lt-1]  < pivot
//	data[lt..gt]   == pivot
//	data[gt+1..hi]  > pivot
func partition3[T any](data []T, lo, hi int, less func(a, b T) bool) (int, int) {
	mid := lo + (hi-lo)/2
	pivotIdx := medianOf3(data, lo, mid, hi, less)
	pivot := data[pivotIdx]

	lt, i, gt := lo, lo, hi
	for i <= gt {
		if less(data[i], pivot) {
			data[lt], data[i] = data[i], data[lt]
			lt++
			i++
		} else if less(pivot, data[i]) {
			data[i], data[gt] = data[gt], data[i]
			gt--
		} else {
			i++
		}
	}
	return lt, gt
}

func medianOf3[T any](data []T, a, b, c int, less func(a, b T) bool) int {
	ab := less(data[a], data[b])
	ac := less(data[a], data[c])
	bc := less(data[b], data[c])

	if ab {
		if bc {
			return b
		}
		if ac {
			return c
		}
		return a
	}

	if !bc {
		return b
	}
	if !ac {
		return c
	}
	return a
}

func insertionSortRange[T any](data []T, lo, hi int, less func(a, b T) bool) {
	for i := lo + 1; i <= hi; i++ {
		x := data[i]
		j := i - 1
		for j >= lo && less(x, data[j]) {
			data[j+1] = data[j]
			j--
		}
		data[j+1] = x
	}
}

// -----------------------------------------------
// Slice — sort.Slice-compatible, index-based less
// -----------------------------------------------

// Slice sorts data using an index-based comparison function,
// matching sort.Slice's signature. Uses quicksort with median-of-3
// pivot and insertion sort for small partitions.
//
// The hybrid merge path is not available here because buffered merging
// requires value-based comparison. For maximum performance on partially
// sorted data, use HybridSort with a value-based less function instead.
func Slice[T any](data []T, less func(i, j int) bool) {
	n := len(data)
	if n < 2 {
		return
	}
	if n <= 16 {
		insertionSortIdx(data, 0, n-1, less)
		return
	}
	qsortIdx(data, 0, n-1, less)
}

func qsortIdx[T any](data []T, lo, hi int, less func(i, j int) bool) {
	for lo < hi {
		if hi-lo <= 16 {
			insertionSortIdx(data, lo, hi, less)
			return
		}

		lt, gt := partition3Idx(data, lo, hi, less)

		if lt-lo < hi-gt {
			qsortIdx(data, lo, lt-1, less)
			lo = gt + 1
		} else {
			qsortIdx(data, gt+1, hi, less)
			hi = lt - 1
		}
	}
}

// partition3Idx performs 3-way partitioning with index-based less.
// The pivot is stored at data[hi] throughout and never moved during the loop,
// so less(i, hi) and less(hi, i) always compare against the pivot value.
func partition3Idx[T any](data []T, lo, hi int, less func(i, j int) bool) (int, int) {
	mid := lo + (hi-lo)/2
	pivotPos := medianOf3Idx(lo, mid, hi, less)
	data[pivotPos], data[hi] = data[hi], data[pivotPos]

	lt, i, gt := lo, lo, hi-1
	for i <= gt {
		if less(i, hi) { // data[i] < pivot
			data[lt], data[i] = data[i], data[lt]
			lt++
			i++
		} else if less(hi, i) { // data[i] > pivot
			data[i], data[gt] = data[gt], data[i]
			gt--
		} else { // equal
			i++
		}
	}
	// Move pivot from hi into the equal range
	data[gt+1], data[hi] = data[hi], data[gt+1]
	return lt, gt + 1
}

func medianOf3Idx(a, b, c int, less func(i, j int) bool) int {
	ab := less(a, b)
	ac := less(a, c)
	bc := less(b, c)

	if ab {
		if bc {
			return b
		}
		if ac {
			return c
		}
		return a
	}

	if !bc {
		return b
	}
	if !ac {
		return c
	}
	return a
}

func insertionSortIdx[T any](data []T, lo, hi int, less func(i, j int) bool) {
	for i := lo + 1; i <= hi; i++ {
		for j := i; j > lo && less(j, j-1); j-- {
			data[j], data[j-1] = data[j-1], data[j]
		}
	}
}

// -----------------------------------------------
// SliceStable — stable sort for small slices
// -----------------------------------------------

// SliceStable sorts data using a stable insertion sort with an index-based
// comparison function, matching sort.SliceStable's signature.
// Optimal for small slices (n < ~50). For larger slices, O(n²) cost dominates.
// Zero allocations.
func SliceStable[T any](data []T, less func(i, j int) bool) {
	n := len(data)
	if n < 2 {
		return
	}
	insertionSortIdx(data, 0, n-1, less)
}

// ------------------------------------
// Buffered merge
// ------------------------------------
//
// Merges sorted ranges data[left:mid] and data[mid:right].
// Copies the smaller half into buf, then merges linearly back into data.
// buf must be at least min(mid-left, right-mid) in length.
func bufferedMerge[T any](data []T, buf []T, left, mid, right int, less func(a, b T) bool) {
	if left >= mid || mid >= right {
		return
	}
	// Already ordered.
	if !less(data[mid], data[mid-1]) {
		return
	}

	leftLen := mid - left
	rightLen := right - mid

	if leftLen <= rightLen {
		// Copy left half into buf, merge left-to-right.
		copy(buf, data[left:mid])
		i, j, k := 0, mid, left
		for i < leftLen && j < right {
			if !less(data[j], buf[i]) {
				data[k] = buf[i]
				i++
			} else {
				data[k] = data[j]
				j++
			}
			k++
		}
		// Remaining buf elements (right-side remainder is already in place).
		copy(data[k:], buf[i:leftLen])
	} else {
		// Copy right half into buf, merge right-to-left.
		copy(buf, data[mid:right])
		i, j, k := leftLen-1, rightLen-1, right-1
		for i >= 0 && j >= 0 {
			if !less(buf[j], data[left+i]) {
				data[k] = buf[j]
				j--
			} else {
				data[k] = data[left+i]
				i--
			}
			k--
		}
		// Remaining buf elements (left-side remainder is already in place).
		copy(data[k-j:], buf[:j+1])
	}
}
