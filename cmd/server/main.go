package main

import (
	"fmt"
	"log"
	"os"

	"github.com/rayning0/ssai/internal/hls"
	"github.com/rayning0/ssai/internal/ssai"
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

	channel := ssai.NewChannelState("wcvb")

	fmt.Printf("Loaded %d HLS snapshots\n", len(snapshots))

	for i, snapshot := range snapshots {
		playlist, err := hls.ParsePlaylist(snapshot.Manifest)
		if err != nil {
			log.Fatalf("parsing snapshot %d: %v", i, err)
		}

		before := len(channel.Segments)

		if err := channel.Ingest(playlist); err != nil {
			log.Fatalf("ingesting snapshot %d: %v", i, err)
		}

		fmt.Printf(
			"Snapshot %2d: received=%2d new=%2d total=%2d\n",
			i,
			len(playlist.Segments),
			len(channel.Segments)-before,
			len(channel.Segments),
		)
	}

	fmt.Println("\nAd breaks:")

	for _, b := range channel.AdBreaks {
		fmt.Printf(
			"ID=%s duration=%.3f observedThrough=%.3f "+
				"remaining=%.3f ended=%t firstObservedSeq=%d\n",
			b.ID,
			b.Duration,
			b.ObservedThrough,
			b.Remaining(),
			b.Ended,
			b.FirstObservedSequence,
		)
	}

	fmt.Printf(
		"\nUnique segments: %d\nLast sequence: %d\n",
		len(channel.Segments),
		channel.LastSequence,
	)
}
