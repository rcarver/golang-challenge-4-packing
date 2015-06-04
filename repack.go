package main

import (
	"fmt"
	"log"
	"sort"
	"sync"
)

const (
	debug = false
)

// A repacker repacks trucks.
type repacker struct {
}

// counter keeps track of stuff going in and out of the warehouse.
type counter struct {
	Name    string
	In, Out int
}

// newCounter initializes a counter with a name.
func newCounter(name string) *counter {
	return &counter{Name: name}
}

// Inc adds one to the "in" count.
func (c *counter) Inc(a int) {
	c.In += a
}

// Dec adds one to the "out" count.
func (c *counter) Dec(a int) {
	c.Out += a
}

// Missing is the difference between in and out.
func (c *counter) Missing() int {
	return c.In - c.Out
}

// String is a nice string describing the counter state.
func (c *counter) String() string {
	return fmt.Sprintf("%s: %d in, %d out (missing %d)", c.Name, c.In, c.Out, c.Missing())
}

// warehouse manages the ins and outs of unpacking and packing.
type warehouse struct {
	trucks        chan truck
	boxes         []box
	hasBox        map[uint32]bool
	boxesMu       sync.Mutex
	palletCounter *counter
	truckCounter  *counter
	boxCounter    *counter
}

func (w *warehouse) addBox(b box) {
	w.boxesMu.Lock()
	w.hasBox[b.id] = true
	w.boxes = append(w.boxes, b)
	w.boxesMu.Unlock()
}

func (w *warehouse) grabSomeBoxes(max int) []box {
	w.boxesMu.Lock()
	if l := len(w.boxes); max > l {
		max = l
	}
	out := make([]box, 0, max)
	for _, b := range w.boxes {
		if w.hasBox[b.id] {
			delete(w.hasBox, b.id)
			out = append(out, b)
			if len(out) > max {
				break
			}
		}
	}
	w.boxesMu.Unlock()
	return out
}

func (w *warehouse) grabAllBoxes() []box {
	w.boxesMu.Lock()
	out := make([]box, 0, len(w.boxes))
	for _, b := range w.boxes {
		if w.hasBox[b.id] {
			delete(w.hasBox, b.id)
			out = append(out, b)
		}
	}
	w.boxesMu.Unlock()
	return out
}

func (w *warehouse) returnBoxes(boxes []box) {
	w.boxesMu.Lock()
	for _, b := range boxes {
		w.hasBox[b.id] = true
	}
	w.boxesMu.Unlock()
}

// Unpack unloads all boxes from the trucks, and parks all of the trucks to be
// re-packed.
func (w *warehouse) Unpack(in <-chan *truck) {
	// Close trucks after consuming everything.
	// Do not close boxes because we need to use it as a buffer later.
	defer close(w.trucks)
	for t := range in {
		emptyTruck := &truck{
			id:      t.id,
			pallets: make([]pallet, 0, len(t.pallets)),
		}
		w.trucks <- *emptyTruck
		w.truckCounter.Inc(1)
		for _, p := range t.pallets {
			w.palletCounter.Inc(1)
			for _, b := range p.boxes {
				w.boxCounter.Inc(1)
				w.addBox(b)
			}
		}
	}
}

// PackTruck re-packs a truck as efficiently as possible.
func (w *warehouse) PackTruck(t *truck) {
	w.truckCounter.Dec(1)
	// Pack up to the truck's pallet capacity.
	for len(t.pallets) < cap(t.pallets) {
		if debug {
			fmt.Printf("Packing truck %d pallet %d\n", t.id, len(t.pallets))
		}
		p := w.packOnePallet()
		// If the pallet comes back empty we're done.
		if len(p.boxes) == 0 {
			return
		}
		w.palletCounter.Dec(1)
		w.boxCounter.Dec(len(p.boxes))
		t.pallets = append(t.pallets, *p)
	}
}

// PackRemainingBoxes puts all remaining boxes onto this last truck, with no
// regard for how many pallets should fit.
func (w *warehouse) PackRemainingBoxes(t *truck) {
	w.truckCounter.Dec(1)
	pallets := w.packAllBoxes()
	for _, p := range pallets {
		w.palletCounter.Dec(1)
		w.boxCounter.Dec(len(p.boxes))
		t.pallets = append(t.pallets, *p)
	}
}

const (
	maxBoxes = 100
)

// packOnePallet pulls boxes from the channel, packs as many as it can onto one
// pallet, then returns any unpacked boxes back to the channel. It returns the
// packed pallet.
func (w *warehouse) packOnePallet() *pallet {
	if debug {
		fmt.Printf("Packing...\n")
	}

	// Pack a pallet.
	pal := &pallet{boxes: make([]box, 0, 16)}
	boxes := w.grabSomeBoxes(maxBoxes)
	unusedBoxes := packWithShelves(pal, boxes)
	w.returnBoxes(unusedBoxes)

	if debug {
		fmt.Printf("Packed %d of %d boxes on pallet\n", len(pal.boxes), len(boxes))
		for _, b := range pal.boxes {
			fmt.Printf("  box %d: x%d, y%d, l%d, w%d\n", b.id, b.x, b.y, b.l, b.w)
		}
		fmt.Printf("%s\n", pal)
	}

	return pal
}

// packAllBoxes pulls all boxes from the channel and packs them onto pallets
// until they are all packed. It returns all of the packed pallets.
func (w *warehouse) packAllBoxes() []*pallet {
	// Pack until all of the boxes are used.
	boxes := w.grabAllBoxes()
	pallets := make([]*pallet, 0, len(boxes))
	for len(boxes) > 0 {
		pal := &pallet{boxes: make([]box, 0, 16)}
		boxes = packWithShelves(pal, boxes)
		pallets = append(pallets, pal)
	}
	return pallets
}

type sortedBoxes []box

func (boxes sortedBoxes) Len() int {
	return len(boxes)
}
func (boxes sortedBoxes) Less(i, j int) bool {
	a, b := boxes[i], boxes[j]
	if a.w == b.w {
		return a.l > b.l
	}
	return a.w > b.w
}
func (boxes sortedBoxes) Swap(i, j int) {
	boxes[i], boxes[j] = boxes[j], boxes[i]
}

// sideways orients the box sideways.
func sideways(b *box) {
	if b.w > b.l {
		b.w, b.l = b.l, b.w
	}
}

// upright orients the box upright.
func upright(b *box) {
	if b.w < b.l {
		b.w, b.l = b.l, b.w
	}
}

// shelf models a horizontal plane of boxes. The height of the shelf is
// determined by the first box. Once a height is set, any additional boxes must
// fit within that height to be added. Boxes can be rotated to fit.
//
// The box coordinate system is very confusing. Here it is:
//
//  ! box x0, y0, w1, l1
//  @ box x1, y0, w1, l3
//
//   (x + l)
//   ^
//
// | !       |  > (y + w)
// | @       |
// | @       |
// | @       |
//
type shelf struct {
	// x starts at zero and changes with each box.
	x uint8
	// y is constant for shelf.
	y uint8
	// w is set by the first box.
	w uint8
	// l is the length of the box.
	l uint8
	// lRemains counts down with each box.
	lRemains uint8
}

// newShelf initializes a new shelf at y position with length.
func newShelf(y, l uint8) *shelf {
	return &shelf{
		x:        0,
		y:        y,
		l:        l,
		lRemains: l,
	}
}

// nextShelf returns a new empty shelf that sits on top of the current. A
// non-zero value for the width sets the size of the shelf.
func (s *shelf) nextShelf(w uint8) *shelf {
	ns := newShelf(s.y+s.w, s.l)
	ns.w = w
	return ns
}

// add puts a box on the shelf if it fits. The box will be rotated to find the
// best placement. If a fit is found, the shelf's positions are updated and
// true is returned. Otherwise false is returned and the shelf is unchanged.
func (s *shelf) add(b *box) bool {
	if s.w == 0 {
		sideways(b)
		s.w = b.w
		s.include(b)
		return true
	}
	upright(b)
	if b.w <= s.w && b.l <= s.lRemains {
		s.include(b)
		return true
	}
	sideways(b)
	if b.w <= s.w && b.l <= s.lRemains {
		s.include(b)
		return true
	}
	return false
}

func (s *shelf) include(b *box) {
	b.x, b.y = s.x, s.y
	s.x += b.l
	s.lRemains -= b.l
}

// packWithShelves fills a pallet with the shelf algorithm, using the boxes given. It
// returns the boxes that were not put onto the pallet.
func packWithShelves(pal *pallet, boxes []box) []box {
	shelf := newShelf(0, palletLength)
	wRemains := uint8(palletWidth)

	sort.Sort(sortedBoxes(boxes))

	//for i, b := range boxes {
	//fmt.Printf("use boxes %d %s\n", i, b)
	//}

	if debug {
		fmt.Printf("  Begin packing...\n")
	}
	i := 0
	for i < len(boxes) {
		b := boxes[i]
		ok := shelf.add(&b)
		if ok {
			if debug {
				fmt.Printf("  + shelf %v, box %v\n", shelf, b)
			}
			i++
			pal.boxes = append(pal.boxes, b)
		} else {
			if debug {
				fmt.Printf("  - shelf %v, box %v\n", shelf, b)
			}
			wRemains -= shelf.w
			if wRemains <= 0 {
				break
			}
			shelf = shelf.nextShelf(wRemains)
		}
	}
	return boxes[i:]
}

func newRepacker(in <-chan *truck, out chan<- *truck) *repacker {
	w := &warehouse{
		trucks:        make(chan truck, 10),
		boxes:         make([]box, 0, 2000),
		hasBox:        make(map[uint32]bool),
		truckCounter:  newCounter("Trucks"),
		palletCounter: newCounter("Pallets"),
		boxCounter:    newCounter("Boxes"),
	}
	go w.Unpack(in)
	go func() {
		for len(w.trucks) < cap(w.trucks) {
			// wait
		}
		// The repacker must close channel out after it detects that
		// channel in is closed so that the driver program will finish
		// and print the stats.
		defer close(out)
		defer func() {
			log.Printf("...\n")
			log.Printf("%s\n", w.truckCounter)
			log.Printf("%s\n", w.palletCounter)
			log.Printf("%s\n", w.boxCounter)
			log.Printf("...\n")
		}()
		for {
			select {
			case t := <-w.trucks:
				if t.id == idLastTruck {
					log.Printf("Packing the last truck...\n")
					w.PackRemainingBoxes(&t)
					out <- &t
					return
				}
				w.PackTruck(&t)
				out <- &t
			}
		}
	}()
	return &repacker{}
}
