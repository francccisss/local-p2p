package test

import (
	"client/protocol"
	"client/utils"
	"fmt"
	"math"
	"testing"
)

var PIECES = [10]string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}

const WINDOW_SIZE = 9

func TestSlidingWindow(t *testing.T) {

	sw := protocol.NewSlidingWindow(WINDOW_SIZE, len(PIECES))

	doneWindow := []string{}

	fmt.Printf("INIT Base: %d, End: %d\n", sw.Base, sw.End)
	for range int(math.Ceil(float64(len(PIECES)) / float64(WINDOW_SIZE))) {

		fmt.Printf("Current window -> Base: %d, End: %d\n", sw.Base, sw.End)
		tmp := []string{}
		// sending

		dif := min(sw.Size, len(PIECES)-sw.Base)
		for range dif {
			tmp = append(tmp, PIECES[sw.Base])
			// ->
			sw.Move()
		}
		fmt.Printf("Dif: %d\n", dif)

		fmt.Println(tmp)

		// receiving
		doneWindow = append(doneWindow, utils.ConcatStr(&tmp))

		fmt.Printf("Next window -> Base: %d, End: %d\n", sw.Base, sw.End)
	}

	fmt.Printf("Sliding Window Result: %+v\n", doneWindow)

}
