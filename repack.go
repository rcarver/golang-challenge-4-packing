package main

// A repacker repacks trucks.
type repacker struct {
}

// This repacker is the worst possible, since it uses a new pallet for
// every box. Your job is to replace it with something better.
func oneBoxPerPallet(t *truck) (out *truck) {
	out = &truck{id: t.id}
	for _, p := range t.pallets {
		for _, b := range p.boxes {
			b.x, b.y = 0, 0
			out.pallets = append(out.pallets, pallet{boxes: []box{b}})
		}
	}
	return
}

func newRepacker(in <-chan *truck, out chan<- *truck) *repacker {
	go func() {
		for t := range in {
			// The last truck is indicated by its id. You might
			// need to do something special here to make sure you
			// send all the boxes.
			if t.id == idLastTruck {
			}

			t = oneBoxPerPallet(t)
			out <- t
		}
		// The repacker must close channel out after it detects that
		// channel in is closed so that the driver program will finish
		// and print the stats.
		close(out)
	}()
	return &repacker{}
}
