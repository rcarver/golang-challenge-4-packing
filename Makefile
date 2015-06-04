run10: build
	./golang-challenge-4-packing < testdata/10trucks.txt

run100: build
	./golang-challenge-4-packing < testdata/100trucks.txt

run100k: golang-challenge-4-packing
	./golang-challenge-4-packing < testdata/100000trucks.txt

build:
	go build .

