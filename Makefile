all: nc

ME:=$(firstword $(MAKEFILE_LIST))

nc: nc.go $(ME)
	go build -o nc
	-goz-util -c nc


clean:
	-@ [ -x nc ] && rm nc

check:
	@echo no checks yet

install:
	mkdir -p $(PREFIX)/bin
	install $(BINS) $(PREFIX)/bin
