package bitfield

import (
	"fmt"
	"math"
)

type BitField struct {
	b         []BitFieldUnit
	UnitCount int
	pieces    int
}

// a BitFieldUnit represents the container of all the 32 bits field of a file
const UNIT_SIZE = 32 // including 0

type BitFieldUnit uint32

const MINIMUM_PIECE = 1

func NewBitField(piecesCount int) BitField {
	BitFieldUnitCount := max(int(math.Ceil(float64(piecesCount)/float64(UNIT_SIZE))), MINIMUM_PIECE)
	bf := BitField{
		b:         make([]BitFieldUnit, BitFieldUnitCount),
		UnitCount: BitFieldUnitCount,
		pieces:    piecesCount - 1, // interest starts at index 0
	}
	fmt.Printf("BitField: %+v\n", bf)
	return bf
}

// `pos` position to shift to
func (bf *BitField) LeftShift(pos int) {
	// pos is the interest that starts at index 0
	// checking it against BitFieldUnitCount makes sure that
	// no access to further reads is possible if > bf.BitFieldUnitCount
	if pos > bf.pieces {
		panic("index overflow")
	}

	t := 0
	for i := range bf.b {
		fmt.Printf("[LEFT SHIFT]-[%d]: %0b\n", i, bf.b[i])
		t += UNIT_SIZE
		if t > pos { // t increments when pos = 32 and t = 32 so now pos is within 64
			base := int(math.Abs(float64(t - UNIT_SIZE))) // [base...]
			dif := pos % UNIT_SIZE                        // [base..dif..max]
			fmt.Printf("[LEFT SHIFT]: shifting within pos: %d and base pos: %d, dif %d\n", t, base, dif)
			bf.b[i] |= 1 << dif
			fmt.Printf("[LEFT SHIFT]-[%d]: %0b\n", i, bf.b[i])
			break
		}
	}
}

func (bf *BitField) CheckBit(pos int) bool {

	if pos > bf.pieces {
		panic("index overflow")
	}
	t := 0
	for i := range bf.b {

		fmt.Printf("[CHECKBIT]-[%d]: %0b\n", i, bf.b[i])
		t += UNIT_SIZE
		if t >= pos {
			base := int(math.Abs(float64(t - UNIT_SIZE))) // [base...]
			dif := pos % UNIT_SIZE                        // [base..dif..max]
			fmt.Printf("[CHECKBIT]:within BitFieldUnit '%d' and base pos: %d, dif %d\n", t, base, dif)
			// returns the value
			if bf.b[i]&(1<<dif) > 0 {
				return true
			}
			fmt.Printf("BitFieldUnit-[%d]: %0b\n", i, bf.b[i])
			return false
		}
	}
	return false

}

const (
	SEC_0 = 1 << iota * 8 // move by byte size
	SEC_1
	SEC_2
	SEC_3
)

// log(n*m), using linear search (slow)
func (bf *BitField) IsFilled() bool {

	fmt.Println("######### Checking units using bitmask ##########")
	for i := range bf.pieces {
		if !bf.CheckBit(i) {
			fmt.Printf("[UNIT-%d] not filled\n", i)
			return false
		}
	}
	fmt.Printf("UNITS filled\n")
	return true
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
	fmt.Println("[PRINT BitFieldUnit START]")
	for i := range bf.b {
		fmt.Printf("BitFieldUnit-[%d]: %0b\n", i, bf.b[i])

	}

	fmt.Println("[PRINT BitFieldUnit END]")

}
