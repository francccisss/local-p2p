package test

import (
	"client/protocol"
	"client/utils/bitfield"
	"fmt"
	"testing"
)

const TEST_FILE = "newfile.webp"
const PATH = "/files/"

func TestDataPieces(t *testing.T) {
	mtd, err := protocol.NewFileMetaData("./files/newfile.webp")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%+v", mtd)
	bf := bitfield.NewBitField(int(mtd.Pieces))
	fmt.Println(len(bf))
	bf.FillBits()

	fr := protocol.FileRequest{Hash: mtd.Hash, Interest: 0}

	if !bf.CheckBit(int(fr.Interest)) {
		t.Fatal("[ TESTING ERROR ]: Bit set to 0")
	}
	// protocol.CreateDataPiece(PATH, fr)
}
