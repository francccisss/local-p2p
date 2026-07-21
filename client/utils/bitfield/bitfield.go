package bitfield

import (
	"fmt"
	"math"
)

type BitField []byte

const BYTE_SIZE = 8
const MINIMUM_PIECE = 1

func NewBitField(piecesCount int) BitField {
	size := max(piecesCount/8, MINIMUM_PIECE)
	bitField := make([]byte, size)
	return bitField
}

// `val` position to shift to
func (b *BitField) LeftShift(val int) {
	t := 0
	for n := range *b {
		t += BYTE_SIZE
		fmt.Printf("t val: %d, shift val: %d\n", t, val)
		if t >= val {
			max := t                                          // [...max]
			base := t - BYTE_SIZE + 1                         // [base...]
			dif := int(math.Abs(float64(max) - float64(val))) // [base..dif..max]
			fmt.Printf("shifting within pos: %d and base pos: %d, dif %d\n", t, base, dif)
			fmt.Printf("Shifted in byte pos: %d\n", n)
			(*b)[n] |= 1 << dif
			break
		}
	}
}

func (b *BitField) CheckBit(pos int) bool {
	t := 0
	for i := range *b {
		t = i * BYTE_SIZE
		if t >= pos {
			max := t                                          // [...max]
			base := t - BYTE_SIZE + 1                         // [base...]
			dif := int(math.Abs(float64(max) - float64(pos))) // [base..dif..max]
			fmt.Printf("within pos: %d and base pos: %d, dif %d\n", t, base, dif)
			fmt.Printf("in byte pos: %d\n", i)
			if (*b)[i]&(1<<dif) == 1 {
				return true
			}
			return false
		}
	}
	return false

}

func (b *BitField) FillBits() {
	fmt.Println("REEEEEEEEEEEEEE")
	for i := range *b {
		for j := range BYTE_SIZE {
			(*b)[i] |= 1 << j
		}
		fmt.Printf("[%d]: %b\n", i, (*b)[i])
	}
}
