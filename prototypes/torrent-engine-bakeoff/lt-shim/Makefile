CXX ?= g++
LT_CFLAGS := $(shell pkg-config --cflags libtorrent-rasterbar)
LT_LIBS := $(shell pkg-config --libs libtorrent-rasterbar)
CXXFLAGS ?= -O2
CXXFLAGS += -std=c++17 -fPIC $(LT_CFLAGS)

all: liblts.a liblts.so

lts.o: lts.cpp lts.h
	$(CXX) $(CXXFLAGS) -c lts.cpp -o $@

liblts.a: lts.o
	ar rcs $@ $^

liblts.so: lts.o
	$(CXX) -shared -o $@ $^ $(LT_LIBS)

clean:
	rm -f lts.o liblts.a liblts.so

.PHONY: all clean
