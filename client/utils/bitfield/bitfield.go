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

// TODO:
// STORING MISSING INFORMATION FROM BITFIELD
// c. set bits per 1 byte's 2^0 position to 1 of each BitFieldUnit
//
//	section3|section2|section1|section0
//
// BitFieldUnit[ 31..25 | 24..16 | 15..8 | 7..0 ]
// each beginning bit of each section per BitFieldUnit
// will be either set to 0 or 1
const (
	SEC_0 = 1 << iota * 8 // move by byte size
	SEC_1
	SEC_2
	SEC_3
)

// reports missing
func (bf *BitField) CheckBitFields() (missing []BitFieldUnit, isFilled bool) {

	// t := 0
	// missingTmp := make([]BitFieldUnit, bf.UnitCount)

	// for each unit, check if
	// every unit is filled by checking if each
	// unit's 32bit value == unint32 max value
	// tmpUnit XOR= 0bFF << 1
	bitMask := ^BitFieldUnit(0)
	fmt.Println("######### Checking units using bitmask ##########")
	for _, unit := range bf.b {
		xorUnit := unit ^ bitMask
		fmt.Printf("[INTEGER]: XOR_UNIT: %d, UNIT: %d\n", xorUnit, unit)
		fmt.Printf("[BITS]: XOR_UNIT: %032b, UNIT: %032b\n", xorUnit, unit)
	}

	return []BitFieldUnit{}, false
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
