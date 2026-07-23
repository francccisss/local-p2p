package test

import (
	"client/utils/bitfield"
	"fmt"
	"testing"
)

const PIECE_COUNT = 500

func TestBitField(t *testing.T) {

	bf := bitfield.NewBitField(PIECE_COUNT)

	for p := range PIECE_COUNT {

		bf.LeftShift(p)
		if !bf.CheckBit(p) {
			t.Fatal("Bit set to 0")
		}
		bf.PrintBitField()

		fmt.Printf("NEXT PIECE\n\n")
	}
}

func TestCheckBitFields(t *testing.T) {

	fmt.Println("TESTING CHECK BITFIELDS")

	bf := bitfield.NewBitField(PIECE_COUNT)
	bf.FillBits()
	bf.PrintBitField()

	if !bf.IsFilled() {
		t.Fatal("Not Filled")
	}

}
