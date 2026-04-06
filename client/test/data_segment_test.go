package test

import (
	_ "client/protocol"
	"client/utils"
	"fmt"
	_ "os"
	"slices"
	"testing"
)

func TestDataSegmentation(t *testing.T) {

	// n := clientProtocol.Node{
	// 	FILE_LOCATION: "/files/",
	// }
	//
	// en, path, err := n.Checkfile("pdd2zwopm2sg1.webp", n.FILE_LOCATION)
	// if err != nil {
	// 	fmt.Println(err)
	// 	t.FailNow()
	// }
	// b, err := os.ReadFile(path + en.Name())
	// if err != nil {
	// 	fmt.Println(err)
	// 	t.FailNow()
	// }

	b := []byte("This is a string where each character is a single byte... i think")
	ds, err := utils.DataSegmentation(b, 10)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	var tmp [][]byte
	for _, d := range ds {
		fmt.Printf("Segment #%d\nData: %+v\n", d.SegmentNum, d)
		tmp = append(tmp, d.DataChunk)
	}

	fmt.Printf("Data len from segment: %d\n", len(slices.Concat(tmp...)))
	fmt.Printf("From byte to string: %s\n", slices.Concat(tmp...))

}
