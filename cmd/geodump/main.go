// Command geodump prints a GEO<area>.DAX map as text (dev tool): walls as
// lines, doors as d (open side) or D (locked), the cell values below.
//
//	geodump DATA AREA BLOCK
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"cotab/dax"
)

func main() {
	if len(os.Args) != 4 {
		log.Fatal("usage: geodump DATA AREA BLOCK")
	}
	l, err := dax.NewLoader(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	block, _ := strconv.Atoi(os.Args[3])
	b := l.Block("GEO"+os.Args[2], block)
	if len(b) < 2+1024 {
		log.Fatalf("no GEO%s block %d", os.Args[2], block)
	}
	g := b[2:]
	wall := func(x, y, dir int) byte {
		i := y*16 + x
		switch dir {
		case 0:
			return g[i] >> 4
		case 2:
			return g[i] & 15
		case 4:
			return g[256+i] >> 4
		}
		return g[256+i] & 15
	}
	door := func(x, y, dir int) byte { return g[768+y*16+x] >> uint(dir) & 3 }
	mark := func(x, y, dir int, open, closed string) string {
		if wall(x, y, dir) == 0 {
			return open
		}
		n := len(closed)
		switch door(x, y, dir) {
		case 1:
			return "dd"[:n]
		case 2, 3:
			return "DD"[:n]
		}
		return closed
	}
	fmt.Print("   ")
	for x := 0; x < 16; x++ {
		fmt.Printf("%2d ", x)
	}
	fmt.Println()
	for y := 0; y < 16; y++ {
		fmt.Print("   ")
		for x := 0; x < 16; x++ {
			fmt.Print("+" + mark(x, y, 0, "  ", "--"))
		}
		fmt.Println("+")
		fmt.Printf("%2d ", y)
		for x := 0; x < 16; x++ {
			fmt.Print(mark(x, y, 6, " ", "|"))
			fmt.Printf("%02x", g[512+y*16+x])
		}
		fmt.Println(mark(15, y, 2, " ", "|"))
	}
}
