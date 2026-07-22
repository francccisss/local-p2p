package test

import (
	"client/utils/bitfield"
	"fmt"
	"testing"
)

func TestBitField(t *testing.T) {
	piecesCount := 200

	bf := bitfield.NewBitField(piecesCount)

	for p := range piecesCount {

		bf.LeftShift(p)
		if !bf.CheckBit(p) {
			t.Fatal("Bit set to 0")
		}
		bf.PrintBitField()

		fmt.Printf("NEXT PIECE\n\n")
	}

}
