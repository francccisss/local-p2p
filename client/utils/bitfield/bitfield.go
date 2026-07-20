package bitfield

import (
	"fmt"
	"math"
)

type BitField []byte

const BIT_SIZE = 8

func NewBitField(piecesCount int) BitField {
	size := piecesCount / 8
	bitField := make([]byte, size)
	return bitField
}

// `val` position to shift to
func (b *BitField) LeftShift(val int) {
	t := 0
	for n := range *b {
		t += BIT_SIZE
		fmt.Printf("t val: %d, shift val: %d\n", t, val)
		if t >= val {
			max := t                                          // [...max]
			base := t - BIT_SIZE + 1                          // [base...]
			dif := int(math.Abs(float64(max) - float64(val))) // [base..dif..max]
			fmt.Printf("shifting within pos: %d and base pos: %d, dif %d\n", t, base, dif)
			fmt.Printf("Shifted in byte pos: %d\n", n)
			(*b)[n] = 1 << dif
			break
		}
	}
}
