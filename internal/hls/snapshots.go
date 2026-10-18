package hls

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type Snapshot struct {
	Timestamp string
	Manifest  string
}

// ReadSnapshots splits a captured curl log into individual
// HLS manifests.
//
// Expected format:
//
// --- 2026-10-17 21:27:17 ---
// #EXTM3U
// ...
// --- 2026-10-17 21:27:28 ---
// #EXTM3U
// ...
func ReadSnapshots(r io.Reader) ([]Snapshot, error) {
	var snapshots []Snapshot
	var current *Snapshot

	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()

		// when we see new header "--- 2026-10-17 21:27:17 ---"
		if strings.HasPrefix(line, "--- ") &&
			strings.HasSuffix(line, " ---") {

			if current != nil {
				snapshots = append(snapshots, *current)
			}

			// start new Snapshot
			// Snapshot{
			// 	Timestamp: "2026-10-17 21:27:17",
			// 	Manifest:  "",
			// }
			current = &Snapshot{
				Timestamp: strings.TrimSuffix(
					strings.TrimPrefix(line, "--- "),
					" ---",
				),
			}

			continue
		}

		if current != nil {
			current.Manifest += line + "\n"
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading snapshots: %w", err)
	}

	if current != nil {
		snapshots = append(snapshots, *current)
	}

	return snapshots, nil
}
