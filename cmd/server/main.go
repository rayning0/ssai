package main

import (
	"fmt"
	"log"
	"os"

	"github.com/rayning0/ssai/internal/hls"
)

func main() {
	file, err := os.Open("testdata/sample-scte-playlist.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	snapshots, err := hls.ReadSnapshots(file)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded %d HLS snapshots\n", len(snapshots))

	for i, snapshot := range snapshots {
		fmt.Printf(
			"Snapshot %d: %s (%d bytes)\n",
			i,
			snapshot.Timestamp,
			len(snapshot.Manifest),
		)
	}
}
