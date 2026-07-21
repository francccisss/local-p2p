package bitfield

import (
	"fmt"
	"math"
)

type BitField struct {
	b         []byte
	byteCount int
	pieces    int
}

const BYTE_COUNT = 8
const MINIMUM_PIECE = 1

func NewBitField(piecesCount int) BitField {
	byteCount := max(piecesCount/8, MINIMUM_PIECE)
	bf := BitField{
		b:         make([]byte, byteCount),
		byteCount: byteCount,
		pieces:    piecesCount,
	}
	fmt.Printf("BitField: %+v\n", bf)
	return bf
}

// `pos` position to shift to
func (bf *BitField) LeftShift(pos int) {
	// pos is the interest that starts at index 0
	// checking it against byteCount makes sure that
	// no access to further reads is possible if > bf.byteCount
	if pos+1 > bf.pieces {
		panic("Index is greater than BitField byteCount")
	}

	t := 0
	for n := range bf.b {
		t += BYTE_COUNT
		fmt.Printf("t pos: %d, shift pos: %d\n", t, pos)
		if t >= pos {
			max := t                                          // [...max]
			base := t - BYTE_COUNT + 1                        // [base...]
			dif := int(math.Abs(float64(max) - float64(pos))) // [base..dif..max]
			fmt.Printf("shifting within pos: %d and base pos: %d, dif %d\n", t, base, dif)
			fmt.Printf("Shifted in byte pos: %d\n", n)
			bf.b[n] |= 1 << dif
			break
		}
	}
}

func (bf *BitField) CheckBit(pos int) bool {

	if pos+1 > bf.pieces {
		panic("Index is greater than BitField byteCount")
	}
	t := 0
	for i := range bf.b {
		t = i * BYTE_COUNT
		if t >= pos {
			max := t                                          // [...max]
			base := t - BYTE_COUNT + 1                        // [base...]
			dif := int(math.Abs(float64(max) - float64(pos))) // [base..dif..max]
			fmt.Printf("within pos: %d and base pos: %d, dif %d\n", t, base, dif)
			fmt.Printf("in byte pos: %d\n", i)
			if bf.b[i]&(1<<dif) == 1 {
				return true
			}
			return false
		}
	}
	return false

}

func (bf *BitField) FillBits() {
	l := 0
	for i := range bf.b {
		for j := range BYTE_COUNT {
			l++
			fmt.Printf("l: %d\n", l)
			if l <= bf.pieces {
				bf.b[i] |= 1 << j
			}
		}
		fmt.Printf("[%d]: %b\n", i, bf.b[i])
	}
}
