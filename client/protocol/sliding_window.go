package protocol

type SlidingWindow struct {
	Base        int
	End         int
	Size        int // windowsize
	contentSize int // content that is being used by the sliding window
}

func NewSlidingWindow(size int, contentSize int) SlidingWindow {
	return SlidingWindow{
		Base:        0,
		End:         size - 1,
		Size:        size,
		contentSize: contentSize,
	}
}

func (s *SlidingWindow) Reset() {
	s.Base = 0
	s.End = 0

}

// increments both base and End
func (s *SlidingWindow) Move() {
	if s.End <= s.contentSize {
		s.End++
	}
	s.Base++
}

// decrements both base and End
func (s *SlidingWindow) Back() {
	s.Base--
	s.End--
}
