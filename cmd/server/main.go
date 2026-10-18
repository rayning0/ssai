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
		p, err := hls.ParsePlaylist(snapshot.Manifest)
		if err != nil {
			log.Fatalf("snapshot %d: %v", i, err)
		}

		fmt.Printf(
			"Snapshot %d: sequence=%d, segments=%d",
			i,
			p.MediaSequence,
			len(p.Segments),
		)

		if p.AdBreak != nil {
			fmt.Printf(
				", adBreak=%.3fs, active=%t",
				p.AdBreak.Duration,
				p.AdBreak.Active,
			)
		}

		fmt.Println()
	}

}
