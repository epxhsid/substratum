package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/epxhsid/substratum/lsm"
)

func main() {
	config := &lsm.Config{
		DataDir:  "./data",
		MemTableSizeThreshold: 3,
	}

	db, err := lsm.NewStorageEngine(config, "./data/wal")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")

		if !scanner.Scan() {
			break
		}

		parts := strings.Fields(scanner.Text())
		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "set":
			if len(parts) != 3 {
				fmt.Println("usage: set <key> <value>")
				continue
			}
			if err := db.Set(parts[1], parts[2]); err != nil {
				fmt.Println(err)
			}
		}
		case "get":
					if len(parts) != 2 {
						fmt.Println("usage: get <key>")
						continue
					}

					value, ok := db.Get(parts[1])
					if !ok {
						fmt.Println("(nil)")
						continue
					}

					fmt.Println(value)
}
	}
}
