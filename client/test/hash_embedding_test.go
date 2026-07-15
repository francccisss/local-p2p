package test

import (
	"client/protocol"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	_ "os"
	"strings"
	"testing"

	"github.com/pkg/xattr"
)

var src = "./files/text.txt"

func TestHashEmbedding(t *testing.T) {
	hash := sha256.New()
	f, err := os.Open(src)
	if err != nil {
		t.Fatal(err.Error())
	}

	_, err = io.Copy(hash, f)
	if err != nil {
		t.Fatal(err.Error())
	}
	hashID := hex.EncodeToString(hash.Sum(nil))

	fmt.Printf("[ TESTING ]: Generated HashID: %s\n", hash)
	err = protocol.EmbbedFileHashID(hashID, src)
	if err != nil {
		t.Fatal(err.Error())
	}
	isSame, returned, err := VerifyEmbeddedHash(src, hashID, protocol.ATTRIBUTE_STRING)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !isSame {
		t.Fatalf("[ TESTING ]: Expected FileHash: %s\nReturned FileHash: %s\n", hashID, returned)
	}
	fmt.Printf("[ TESTING ]: FileHash: '%s' Embedded into '%s'\n", returned, src)
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
