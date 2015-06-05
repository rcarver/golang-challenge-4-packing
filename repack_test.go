package main

import (
	"sort"
	"testing"
)

func Test_sortedBoxes(t *testing.T) {
	a := box{0, 0, 5, 1, 91}
	b := box{0, 0, 5, 2, 90}
	c := box{0, 0, 4, 2, 92}
	d := box{0, 0, 3, 2, 93}
	gotBoxes := []box{c, b, d, a}
	wantBoxes := []box{a, b, c, d}
	sort.Sort(sortedBoxes(gotBoxes))
	for i := range make([]struct{}, 4) {
		if got, want := gotBoxes[i], wantBoxes[i]; got != want {
			t.Errorf("%d got %v, want %v", i, got, want)
		}
	}
}

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
	want := shelf{0, 1, 0, 4, 4}
	if *s != want {
		t.Errorf("got %v, want %v", s, want)
	}
}

func Test_shelf_nextShelf(t *testing.T) {
	s := newShelf(1, 7)
	s.add(&box{0, 0, 4, 2, 99})
	wantNow := shelf{4, 1, 2, 7, 3}
	if *s != wantNow {
		t.Errorf("got now: %v, want %v", s, wantNow)
	}
	tests := []struct {
		gotShelf  shelf
		wantShelf shelf
	}{
		{
			*s.nextShelf(0),
			shelf{0, 3, 0, 7, 7},
		},
		{
			*s.nextShelf(3),
			shelf{0, 3, 3, 7, 7},
		},
	}
	for i, test := range tests {
		if got, want := test.gotShelf, test.wantShelf; got != want {
			t.Errorf("%d nextShelf got %v, want %v", i, got, want)
		}
	}
}

func Test_shelf_add(t *testing.T) {
	s := newShelf(1, 9)

	tests := []struct {
		boxIn     box
		boxOut    box
		shelf     shelf
		sX        uint8
		sLRemains uint8
		ok        bool
	}{
		{
			// First box is sideways.
			boxIn:  box{0, 0, 3, 2, 99},
			boxOut: box{0, 1, 2, 3, 99},
			shelf:  shelf{3, 1, 2, 9, 6},
			ok:     true,
		},
		{
			// This box fits upright.
			boxIn:  box{0, 0, 1, 2, 99},
			boxOut: box{3, 1, 2, 1, 99},
			shelf:  shelf{4, 1, 2, 9, 5},
			ok:     true,
		},
		{
			// This box fits sideways.
			boxIn:  box{0, 0, 4, 2, 99},
			boxOut: box{4, 1, 2, 4, 99},
			shelf:  shelf{8, 1, 2, 9, 1},
			ok:     true,
		},
		{
			// This box does not fit.
			boxIn:  box{0, 0, 1, 3, 99},
			boxOut: box{0, 0, 1, 3, 99},
			shelf:  shelf{8, 1, 2, 9, 1},
			ok:     false,
		},
	}
	for i, test := range tests {
		ok := s.add(&test.boxIn)
		if got, want := ok, test.ok; got != want {
			t.Errorf("%d: ok: got %v, want %v", i, got, want)
			continue
		}
		if got, want := *s, test.shelf; got != want {
			t.Errorf("%d: shelf: got %s, want %s", i, got, want)
		}
		if test.boxIn != test.boxOut {
			t.Errorf("%d: box: got %s, want %s", i, test.boxIn, test.boxOut)
		}
	}
}

func Test_shelf_include(t *testing.T) {
	s := newShelf(1, 4)
	b := box{0, 0, 3, 2, 99}
	s.include(&b)
	if got, want := s.x, uint8(2); got != want {
		t.Errorf("shelf.x got %d, want %d", got, want)
	}
	if got, want := s.y, uint8(1); got != want {
		t.Errorf("shelf.y got %d, want %d", got, want)
	}
	if got, want := s.lRemains, uint8(2); got != want {
		t.Errorf("shelf.lRemains got %d, want %d", got, want)
	}
	if got, want := b.x, uint8(0); got != want {
		t.Errorf("box.x got %d, want %d", got, want)
	}
	if got, want := b.y, uint8(1); got != want {
		t.Errorf("box.y got %d, want %d", got, want)
	}
}

func Test_packPallet(t *testing.T) {
	boxes := make([]box, 0)
	w := warehouse{boxes: boxes}
	boxes = append(boxes, box{0, 0, 2, 1, 90})
	boxes = append(boxes, box{0, 0, 1, 1, 91})
	boxes = append(boxes, box{0, 0, 1, 3, 92})
	boxes = append(boxes, box{0, 0, 2, 1, 93})
	boxes = append(boxes, box{0, 0, 1, 1, 94})
	pal := w.packOnePallet()
	if err := pal.IsValid(); err != nil {
		t.Fatalf("Pallet is not packed correctly: %s", err)
	}
}
