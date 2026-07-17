package test

import (
	"client/protocol"
	"fmt"
	"strings"
	"testing"

	"github.com/pkg/xattr"
)

var src = "./files/text.txt"

func TestHashEmbedding(t *testing.T) {
	metaData, err := protocol.NewFileMetaData(src)
	if err != nil {
		t.Fatal(err.Error())

	}

	isSame, returned, err := VerifyEmbeddedHash(src, metaData.Hash, protocol.ATTRIBUTE_STRING)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !isSame {
		t.Fatalf("[ TESTING ]: Expected FileHash: %s\nReturned FileHash: %s\n", metaData.Hash, returned)
	}
	fmt.Printf("[ TESTING ]: Created new FILE META DATA\n%+v\n", metaData)
}

func VerifyEmbeddedHash(src string, hashID protocol.FileHash, key string) (bool, string, error) {

	b, err := xattr.Get(src, key)
	if err != nil {
		return false, "", err
	}
	if strings.Compare(string(b), hashID) != 0 {
		return false, "", nil
	}
	return true, string(b), nil

}
