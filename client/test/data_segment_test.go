package test

import (
	"bytes"
	"client/protocol"
	"fmt"
	"math"
	"os"
	"testing"
)

func TestDataPieces(t *testing.T) {

	n := protocol.Node{
		FILE_LOCATION: "/files/",
	}

	// pre processing file
	testFile := "IosevkaTerm.zip"
	en, path, err := protocol.Checkfile(testFile, n.FILE_LOCATION)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	b, err := os.ReadFile(path + en.Name())
	fmt.Printf("file len: %d\n", len(b))

	if err != nil {
		fmt.Println("Error Reading file for len")
		t.FailNow()
	}

	finfo, err := en.Info()
	if err != nil {
		fmt.Println("Error Opening file through syscall")
		t.FailNow()
	}
	// sent by peer to request file
	dfinfo := protocol.FileRequest{
		Hash:      testFile,
		Size:      finfo.Size(),
		Offset:    0,
		BlockSize: int64(os.Getpagesize() * 256),
	}

	// this is all set from reading the meta file from DHT Server
	dfinfo.Pieces = int64(math.Ceil(float64(dfinfo.Size) / float64(dfinfo.BlockSize)))

	fmt.Printf("Chunk size - %d: Bytes, %d: Mega Bytes\n", int(dfinfo.BlockSize), int(dfinfo.BlockSize/1024))

	fmt.Printf("Total Pieces to create: %d\n", dfinfo.Pieces)

	var db bytes.Buffer

	// loop represents the call to the peer for each pieces
	for range dfinfo.Pieces {
		fmt.Printf("Piece Pos: %d\n", dfinfo.Offset)
		fmt.Printf("Piece Size: %d\n", dfinfo.BlockSize)

		ds, headerLen, err := protocol.CreateDataPiece(path+en.Name(), &dfinfo)

		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}
		dsBuf := ds.Bytes()
		sh, n, err := protocol.ParsePieceHeader(dsBuf[:headerLen])
		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}

		fmt.Println("--------------------------------------------------")
		fmt.Printf("Cluster Name: %s\n", sh.ClusterName)
		fmt.Printf("Piece Position: %d\n", sh.PiecePosition)
		fmt.Printf("Piece Size: %d\n", sh.PieceSize)
		fmt.Printf("Total Pieces: %d\n", sh.TotalPieces)
		fmt.Printf("Payload Size: %d\n", len(dsBuf[n:]))
		db.Write(dsBuf[n:])
		fmt.Println("--------------------------------------------------")
	}

	err = os.WriteFile("./tmp/Iozevka.zip", db.Bytes(), 0644)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

}
