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
	mtd, err := protocol.NewFileMetaData("./files/Advanced.Programming.in.the.UNIX.Environment.3rd.Edition.0321637739.pdf")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%+v", mtd)
	bf := bitfield.NewBitField(int(mtd.Pieces))

	fr := protocol.FileRequest{Hash: mtd.Hash, Interest: 2}

	bf.LeftShift(fr.Interest)
	bf.PrintBitField()
	if !bf.CheckBit(int(fr.Interest)) {
		t.Fatal("[ TESTING ERROR ]: Bit set to 0")
	}
	entry, path, err := protocol.Checkfile(mtd.Hash, PATH)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("path: %s. entry: %s\n", path, entry)
	b, err := protocol.CreateDataPiece(path+entry.Name(), fr)
	if err != nil {
		t.Fatal(err)
	}
	pieceHeader, err := protocol.ParsePieceHeader(b)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Piece Header: %+v\n", pieceHeader)
}
