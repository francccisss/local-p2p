package bitfield

import (
	"fmt"
	"math"
)

type BitField struct {
	b         []int
	unitCount int
	pieces    int
}

// a unit represents the container of all the 32 bits field of a file
const UNIT_SIZE = 32
const MINIMUM_PIECE = 1

func NewBitField(piecesCount int) BitField {
	unit := max(int(math.Ceil(float64(piecesCount)/float64(UNIT_SIZE))), MINIMUM_PIECE)
	bf := BitField{
		b:         make([]int, unit),
		unitCount: unit,
		pieces:    piecesCount,
	}
	fmt.Printf("BitField: %+v\n", bf)
	return bf
}

// `pos` position to shift to
func (bf *BitField) LeftShift(pos int) {
	// pos is the interest that starts at index 0
	// checking it against unitCount makes sure that
	// no access to further reads is possible if > bf.unitCount
	if pos+1 > bf.pieces {
		panic("Index is greater than BitField unitCount")
	}

	t := 0
	for n := range bf.b {
		t += UNIT_SIZE
		fmt.Printf("t pos: %d, shift pos: %d\n", t, pos)
		if t >= pos {
			max := t                                          // [...max]
			base := t - UNIT_SIZE + 1                         // [base...]
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
		panic("Index is greater than BitField unitCount")
	}
	t := 0
	for i := range bf.b {
		t = i * UNIT_SIZE
		if t >= pos {
			max := t                                          // [...max]
			base := t - UNIT_SIZE + 1                         // [base...]
			dif := int(math.Abs(float64(max) - float64(pos))) // [base..dif..max]
			fmt.Printf("within pos: %d and base pos: %d, dif %d\n", t, base, dif)
			fmt.Printf("in int pos: %d\n", i)
			if bf.b[i]&(1<<dif) == 1 {
				return true
			}
			return false
		}
	}
	return false

}

// reports missing
func (bf *BitField) CheckBitFields() (missing []int, isFilled bool) {

	// t := 0
	// missingTmp := make([]int, bf.unitCount)
	// for i := range bf.b {
	//
	// }
	return []int{}, false
}

func (bf *BitField) FillBits() {
	l := 0
	for i := range bf.b {
		for j := range UNIT_SIZE {
			l++
			if l <= bf.pieces {
				bf.b[i] |= 1 << j
			}
		}
	}
}
func (bf *BitField) PrintBitField() {
	for i := range bf.b {
		fmt.Printf("[%d]: %0b\n", i, bf.b[i])

	}

}
