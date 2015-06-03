package main

import "testing"

func TestOrientations(t *testing.T) {
	tests := []struct {
		have         box
		wantUpright  box
		wantSideways box
	}{
		{
			have:         box{0, 0, 3, 5, 99},
			wantUpright:  box{0, 0, 5, 3, 99},
			wantSideways: box{0, 0, 3, 5, 99},
		},
		{
			have:         box{0, 0, 5, 3, 99},
			wantUpright:  box{0, 0, 5, 3, 99},
			wantSideways: box{0, 0, 3, 5, 99},
		},
		{
			have:         box{0, 0, 3, 3, 99},
			wantUpright:  box{0, 0, 3, 3, 99},
			wantSideways: box{0, 0, 3, 3, 99},
		},
	}
	for _, test := range tests {
		upright(&test.have)
		if test.have != test.wantUpright {
			t.Errorf("upright  got %s, want %s", test.have, test.wantUpright)
		}
		sideways(&test.have)
		if test.have != test.wantSideways {
			t.Errorf("sideways got %s, want %s", test.have, test.wantSideways)
		}
	}

}

func Test_newShelf(t *testing.T) {
	s := newShelf(1, 4)
	if got, want := s.x, uint8(1); got != want {
		t.Errorf("shelf.x got %d, want %d", got, want)
	}
	if got, want := s.y, uint8(0); got != want {
		t.Errorf("shelf.y got %d, want %d", got, want)
	}
	if got, want := s.w, uint8(0); got != want {
		t.Errorf("shelf.w got %d, want %d", got, want)
	}
	if got, want := s.lRemains, uint8(4); got != want {
		t.Errorf("shelf.lRemains got %d, want %d", got, want)
	}
}

func Test_shelf_add(t *testing.T) {
	s := newShelf(1, 9)

	tests := []struct {
		boxIn     box
		boxOut    box
		sX        uint8
		sLRemains uint8
		ok        bool
	}{
		{
			// First box is sideways.
			boxIn:     box{0, 0, 3, 2, 99},
			boxOut:    box{1, 0, 2, 3, 99},
			sX:        4,
			sLRemains: 6,
			ok:        true,
		},
		{
			// This box fits upright.
			boxIn:     box{0, 0, 1, 2, 99},
			boxOut:    box{4, 0, 2, 1, 99},
			sX:        5,
			sLRemains: 5,
			ok:        true,
		},
		{
			// This box fits sideways.
			boxIn:     box{0, 0, 4, 2, 99},
			boxOut:    box{5, 0, 2, 4, 99},
			sX:        9,
			sLRemains: 1,
			ok:        true,
		},
		{
			// This box does not fit.
			boxIn:     box{0, 0, 1, 3, 99},
			boxOut:    box{0, 0, 1, 3, 99},
			sX:        9,
			sLRemains: 1,
			ok:        false,
		},
	}
	for i, test := range tests {
		ok := s.add(&test.boxIn)
		if got, want := ok, test.ok; got != want {
			t.Errorf("%d: ok: got %v, want %v", i, got, want)
			continue
		}
		if got, want := s.w, uint8(2); got != want {
			t.Errorf("%d shelf.w: got %d, want %d", i, got, want)
		}
		if test.boxIn != test.boxOut {
			t.Errorf("%d: box: got %s, want %s", i, test.boxIn, test.boxOut)
		}
		if got, want := s.x, test.sX; got != want {
			t.Errorf("%d: shelf.x: got %d, want %d", i, got, want)
		}
		if got, want := s.lRemains, test.sLRemains; got != want {
			t.Errorf("%d: shelf.lRemains: got %d, want %d", i, got, want)
		}
	}
}

func Test_shelf_include(t *testing.T) {
	s := newShelf(1, 4)
	b := box{0, 0, 3, 2, 99}
	s.include(&b)
	if got, want := s.x, uint8(3); got != want {
		t.Errorf("shelf.x got %d, want %d", got, want)
	}
	if got, want := s.lRemains, uint8(2); got != want {
		t.Errorf("shelf.lRemains got %d, want %d", got, want)
	}
	if got, want := b.x, uint8(1); got != want {
		t.Errorf("box.x got %d, want %d", got, want)
	}
	if got, want := b.y, uint8(0); got != want {
		t.Errorf("box.y got %d, want %d", got, want)
	}
}
