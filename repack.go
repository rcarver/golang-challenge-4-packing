package main

import (
	"fmt"
	"log"
	"sync"
)

var debug = false

// A repacker repacks trucks.
type repacker struct {
}

// A pallet is 4x4 units, 16 units total.

type packingBuffer struct {
	tMu      sync.Mutex
	trucks   []*truck
	bMu      sync.Mutex
	boxes    []*box
	trucksCh chan *truck
	boxesCh  chan *box
}

func (buf *packingBuffer) addTruck(t truck) {
	out := &truck{
		id:      t.id,
		pallets: make([]pallet, 0, len(t.pallets)),
	}
	buf.tMu.Lock()
	buf.trucks = append(buf.trucks, out)
	buf.tMu.Unlock()
	buf.trucksCh <- out
}

func (buf *packingBuffer) addBox(b box) {
	//buf.tMu.Lock()
	//buf.boxes = append(buf.boxes, &b)
	//buf.tMu.Unlock()
	go func() { buf.boxesCh <- &b }()
}

func accumulate(buf *packingBuffer, t *truck) {
	//log.Printf("Input truck %d with %d pallets\n", t.id, len(t.pallets))
	for _, p := range t.pallets {
		//log.Printf("Input truck %d, pallet %d/%d", t.id, pi+1, len(t.pallets))
		//log.Printf("%s\n", p)
		for _, b := range p.boxes {
			//log.Printf("Input box %d, %d/%d\n", b.id, bi+1, len(p.boxes))
			buf.addBox(b)
		}
	}
	buf.addTruck(*t)
}

func pack(buf *packingBuffer, out chan<- *truck) {
	for {
		select {
		case t := <-buf.trucksCh:
			if debug {
				log.Printf("Output truck %d with %d pallets\n", t.id, cap(t.pallets))
			}
			packTruck(buf, t)
			if debug {
				log.Printf("Packed truck %d with %d pallets\n", t.id, len(t.pallets))
				for i, p := range t.pallets {
					log.Printf("Truck %d Pallet %d/%d", t.id, i+1, len(t.pallets))
					log.Printf("%s\n", p)
				}
			}
			out <- t
		}
	}
}

func packTruck(buf *packingBuffer, t *truck) {
	for {
		p := &pallet{boxes: make([]box, 0, 16)}
		fmt.Printf("Packing truck %d\n", t.id)
		packPallet(buf, p)
		fmt.Printf("-> %d boxes in pallet\n", len(p.boxes))
		fmt.Printf("%s\n", p)
		t.pallets = append(t.pallets, *p)

		if len(t.pallets) == cap(t.pallets) {
			if debug {
				log.Printf("Truck %d has all pallets", t.id)
			}
			return
		}
	}
}

var maxFails = 20

func packPallet(buf *packingBuffer, p *pallet) {
	fails := 0
	for b := range buf.boxesCh {
		ok := false

		if len(p.boxes) == 0 {
			b.x, b.y = 0, 0
			ok = true
		}
		if len(p.boxes) == 1 {
			l := p.boxes[0].x + p.boxes[0].l + b.l
			if l < 4 {
				b.x = p.boxes[0].x + p.boxes[0].l
				b.y = 0
				ok = true
			}
		}

		if ok {
			p.boxes = append(p.boxes, *b)
		} else {
			go func(bb box) { buf.addBox(bb) }(*b)
			fails++
			if fails > maxFails {
				return
			}
		}
	}
}

func newRepacker(in <-chan *truck, out chan<- *truck) *repacker {
	buf := &packingBuffer{
		trucks:   make([]*truck, 0, 10000),
		boxes:    make([]*box, 0, 10000),
		trucksCh: make(chan *truck),
		boxesCh:  make(chan *box),
	}

	go func() {
		pack(buf, out)
		// The repacker must close channel out after it detects that
		// channel in is closed so that the driver program will finish
		// and print the stats.
		//log.Printf("Closing output...\n")
		close(out)
	}()
	go func() {
		for t := range in {
			accumulate(buf, t)
			// The last truck is indicated by its id. You might
			// need to do something special here to make sure you
			// send all the boxes.
			if t.id == idLastTruck {
				log.Printf("Last truck...\n")
			}
		}
	}()
	return &repacker{}
}
