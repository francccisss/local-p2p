package test

import (
	"client/protocol"
	"fmt"
	"testing"
)

var src = "./files/new_dir/text.txt"

func TestHashEmbedding(t *testing.T) {
	metaData, err := protocol.NewFileMetaData(src)
	if err != nil {
		t.Fatal(err.Error())

	}

	isSame, returned, err := protocol.VerifyEmbeddedHash(src, metaData.Hash, protocol.ATTRIBUTE_STRING)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !isSame {
		t.Fatalf("[ TESTING ]: Expected FileHash: %s\nReturned FileHash: %s\n", metaData.Hash, returned)
	}
	fmt.Printf("[ TESTING ]: Created new FILE META DATA\n%+v\n", metaData)
}
