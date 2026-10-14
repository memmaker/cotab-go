// Command ecldump counts the script commands in the ECL<n>.DAX blocks
// (dev tool: a linear sweep, so data between commands can miscount).
//
//	ecldump DATA [AREA]
package main

import (
	"fmt"
	"log"
	"os"
	"sort"

	"cotab/dax"
)

// operands per command (O04C:1FD5 parse_command), -1 for the ones whose
// count follows in their operands.
var operands = map[byte]int{
	1: 1, 2: 1, 10: 1, 14: 1, 17: 1, 18: 1, 29: 1, 32: 1, 45: 1, 50: 1, 52: 1, 54: 1, 56: 1, 57: 1, 60: 1, 63: 1, 64: 1,
	3: 2, 8: 2, 9: 2, 15: 2, 16: 2, 31: 2, 34: 2,
	4: 3, 5: 3, 6: 3, 7: 3, 11: 3, 12: 3, 33: 3, 40: 3, 42: 3, 47: 3, 48: 3, 53: 3, 55: 3, 59: 3,
	20: 4, 35: 4, 46: 5, 30: 6, 44: 6, 39: 8, 41: 14,
	21: 3, 43: 2, 37: 2, 38: 2, // counted items follow
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: ecldump DATA [AREA]")
	}
	l, err := dax.NewLoader(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	area := "2"
	if len(os.Args) > 2 {
		area = os.Args[2]
	}
	counts := map[byte]int{}
	for id := 0; id < 256; id++ {
		b := l.Block("ECL"+area, id)
		if len(b) < 2 {
			continue
		}
		code := b[2:]
		pc := 0
		for pc < len(code) {
			c := code[pc]
			counts[c]++
			n, ok := operands[c]
			if !ok {
				pc++
				continue
			}
			p := pc
			for k := 0; k < n && p+2 < len(code); k++ {
				t := code[p+1]
				p += 2
				if t == 1 || t == 2 || t == 3 || t == 129 {
					p++
				} else if t == 128 {
					p += int(code[p])
				}
			}
			pc = p + 1
		}
	}
	keys := make([]int, 0, len(counts))
	for k := range counts {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, k := range keys {
		fmt.Printf("%3d %6d\n", k, counts[byte(k)])
	}
}
