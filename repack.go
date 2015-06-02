package main

import (
	"fmt"
	"log"
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
func (c *counter) Inc() {
	c.In++
}

// Dec adds one to the "out" count.
func (c *counter) Dec() {
	c.Out++
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
	boxes         chan box
	palletCounter *counter
	truckCounter  *counter
	boxCounter    *counter
}

// Unpack unloads all boxes from the trucks, and parks all of the trucks to be
// re-packed.
func (w *warehouse) Unpack(in <-chan *truck) {
	defer close(w.trucks)
	//defer close(w.boxes)
	for t := range in {
		emptyTruck := &truck{
			id:      t.id,
			pallets: make([]pallet, 0, len(t.pallets)),
		}
		w.trucks <- *emptyTruck
		w.truckCounter.Inc()
		for _, p := range t.pallets {
			w.palletCounter.Inc()
			for _, b := range p.boxes {
				w.boxCounter.Inc()
				w.boxes <- b
			}
		}
	}
}

// PackTruck re-packs a truck as efficiently as possible.
func (w *warehouse) PackTruck(t *truck) {
	w.truckCounter.Dec()
	for len(t.pallets) < cap(t.pallets) {
		p := &pallet{
			boxes: make([]box, 0, 16),
		}
		packPallet(p, w)
		w.palletCounter.Dec()
		t.pallets = append(t.pallets, *p)
	}
}

// PackRemainingBoxes puts all boxes onto this last truck, with no regard for
// efficiency.
func (w *warehouse) PackRemainingBoxes(t *truck) {
	close(w.boxes)
	w.truckCounter.Dec()
	for b := range w.boxes {
		p := &pallet{
			boxes: []box{b},
		}
		w.boxCounter.Dec()
		w.palletCounter.Dec()
		t.pallets = append(t.pallets, *p)
	}
}

func chooseBox(boxes []*box, usedBoxes map[uint32]bool) *box {
	for _, b := range boxes {
		if !usedBoxes[b.id] {
			return b
		}
	}
	return nil
}

var (
	maxBoxes = 64
	maxFails = 64
)

func packPallet(p *pallet, w *warehouse) {
	boxes := make([]*box, 0, maxBoxes)
	usedBoxes := make(map[uint32]bool)

	for len(boxes) < cap(boxes) {
		select {
		case b := <-w.boxes:
			b.x, b.y = 0, 0
			boxes = append(boxes, &b)
		}
	}

	defer func() {
		for _, b := range boxes {
			if !usedBoxes[b.id] {
				w.boxes <- *b
			}
		}
	}()

	fails := 0
	for {
		ok := false
		b := chooseBox(boxes, usedBoxes)
		if b == nil {
			return
		}

		if len(p.boxes) == 0 {
			ok = true
		}
		if len(p.boxes) > 0 && len(p.boxes) <= 4 {
			pb := p.boxes[len(p.boxes)-1]
			l := pb.x + pb.l + b.l
			if l < 4 {
				b.x = pb.x + pb.l
				b.y = 0
				ok = true
			}
		}

		if ok {
			w.boxCounter.Dec()
			usedBoxes[b.id] = true
			p.boxes = append(p.boxes, *b)
		} else {
			fails++
			if fails < maxFails {
				return
			}
		}
	}
}

func newRepacker(in <-chan *truck, out chan<- *truck) *repacker {
	w := &warehouse{
		trucks:        make(chan truck, 200),
		boxes:         make(chan box, 2000),
		truckCounter:  newCounter("Trucks"),
		palletCounter: newCounter("Pallets"),
		boxCounter:    newCounter("Boxes"),
	}
	go w.Unpack(in)
	go func() {
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
