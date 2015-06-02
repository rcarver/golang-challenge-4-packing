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
		w.truckCounter.Inc(1)
		for _, p := range t.pallets {
			w.palletCounter.Inc(1)
			for _, b := range p.boxes {
				w.boxCounter.Inc(1)
				w.boxes <- b
			}
		}
	}
}

// PackTruck re-packs a truck as efficiently as possible.
func (w *warehouse) PackTruck(t *truck) {
	w.truckCounter.Dec(1)
	for len(t.pallets) < cap(t.pallets) {
		p := packOnePallet(w.boxes)
		w.palletCounter.Dec(1)
		w.boxCounter.Dec(len(p.boxes))
		t.pallets = append(t.pallets, *p)
	}
}

// PackRemainingBoxes puts all boxes onto this last truck, with no regard for
// efficiency.
func (w *warehouse) PackRemainingBoxes(t *truck) {
	close(w.boxes)
	w.truckCounter.Dec(1)
	pallets := packAllBoxes(w.boxes)
	for _, p := range pallets {
		w.palletCounter.Dec(1)
		w.boxCounter.Dec(len(p.boxes))
		t.pallets = append(t.pallets, *p)
	}
}

func packOnePallet(boxes chan box) *pallet {
	var boxCap int
	if maxBoxes > len(boxes) {
		boxCap = len(boxes)
	} else {
		boxCap = maxBoxes
	}
	packer := &palletPacker{
		boxes:     make([]*box, 0, boxCap),
		usedBoxes: make(map[uint32]bool),
	}
	packer.takeBoxes(boxes)
	defer packer.putBackUnusedBoxes(boxes)
	p := &pallet{
		boxes: make([]box, 0, 16),
	}
	packer.pack(p)
	return p
}

func packAllBoxes(boxes chan box) []*pallet {
	packer := &palletPacker{
		boxes:     make([]*box, 0, len(boxes)),
		usedBoxes: make(map[uint32]bool),
	}
	packer.takeBoxes(boxes)
	pallets := make([]*pallet, 0, len(packer.boxes))
	for len(packer.usedBoxes) < len(packer.boxes) {
		p := &pallet{
			boxes: make([]box, 0, 16),
		}
		packer.pack(p)
		pallets = append(pallets, p)
	}
	return pallets
}

var (
	maxBoxes = 64
	maxFails = 64
)

type palletPacker struct {
	boxes     []*box
	usedBoxes map[uint32]bool
}

func (p *palletPacker) takeBoxes(boxes <-chan box) {
	for len(p.boxes) < cap(p.boxes) {
		select {
		case b := <-boxes:
			b.x, b.y = 0, 0
			p.boxes = append(p.boxes, &b)
		}
	}
}
func (p *palletPacker) putBackUnusedBoxes(boxes chan<- box) {
	for _, b := range p.boxes {
		if !p.usedBoxes[b.id] {
			boxes <- *b
		}
	}
}

func (p *palletPacker) nextBox() *box {
	for _, b := range p.boxes {
		if !p.usedBoxes[b.id] {
			return b
		}
	}
	return nil
}
func (p *palletPacker) pack(pal *pallet) {
	fails := 0
	for {
		ok := false
		b := p.nextBox()
		if b == nil {
			return
		}

		if len(pal.boxes) == 0 {
			ok = true
		}
		if len(pal.boxes) > 0 && len(pal.boxes) <= 4 {
			pb := pal.boxes[len(pal.boxes)-1]
			l := pb.x + pb.l + b.l
			if l < 4 {
				b.x = pb.x + pb.l
				b.y = 0
				ok = true
			}
		}

		if ok {
			p.usedBoxes[b.id] = true
			pal.boxes = append(pal.boxes, *b)
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
