package test

import (
	clientProtocol "client/protocol"
	"client/utils"
	"fmt"
	"io/fs"
	"os"
	_ "os"
	"slices"
	"testing"
)

func TestDataSegmentation(t *testing.T) {

	n := clientProtocol.Node{
		FILE_LOCATION: "/files/",
	}

	en, path, err := n.Checkfile("pdd2zwopm2sg1.webp", n.FILE_LOCATION)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	b, err := os.ReadFile(path + en.Name())
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

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
	conctData := slices.Concat(tmp...)
	fmt.Printf("Data len from segment: %d\n", len(conctData))
	fmt.Printf("From byte to string: %s\n", conctData)

	var fm fs.FileMode

	fm |= fs.ModePerm

	err = os.WriteFile("newfile.webp", conctData, fm)

	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
}
