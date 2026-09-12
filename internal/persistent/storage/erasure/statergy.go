package erasure

import (
	"math"

	"github.com/avatar31/halmidi/internal/persistent/storage/disk"
)

// ErasureSet represents a single erasure coding set configuration
type ErasureSet struct {
	DataShards   int32
	ParityShards int32
	TotalShards  int32
	Disks        []disk.Disk // Which disks belong to this set
}

// Strategy represents the erasure coding distribution strategy across disks
type Strategy struct {
	Sets []ErasureSet
}

// CalculateOptimalStrategy determines the best erasure coding configuration for given disks
// It aims to maximize data reliability while maintaining efficiency
func CalculateOptimalStrategy(diskMap map[int]disk.Disk) Strategy {
	strategy := Strategy{
		Sets: make([]ErasureSet, 0),
	}
	totalDisks := int32(len(diskMap))
	// For small disk counts (6-16), use single set
	if totalDisks <= 16 {
		dataShards, parityShards := calculateShardDistribution(totalDisks)
		strategy.Sets = append(strategy.Sets, ErasureSet{
			DataShards:   dataShards,
			ParityShards: parityShards,
			TotalShards:  totalDisks,
			Disks:        makeRange(0, totalDisks, diskMap),
		})
		return strategy
	}

	// For larger disk counts, create multiple sets
	// Use 25-30% parity ratio for better reliability
	remainingDisks := totalDisks
	currentIndex := int32(0)

	for remainingDisks >= 6 {
		setSize := calculateOptimalSetSize(remainingDisks)
		dataShards, parityShards := calculateShardDistribution(setSize)

		strategy.Sets = append(strategy.Sets, ErasureSet{
			DataShards:   dataShards,
			ParityShards: parityShards,
			TotalShards:  setSize,
			Disks:        makeRange(currentIndex, currentIndex+setSize, diskMap),
		})

		remainingDisks -= setSize
		currentIndex += setSize
	}

	// If we have remaining disks (< 6), distribute them to existing sets
	if remainingDisks > 0 {
		// Add remaining disks as additional parity to the last set
		lastSet := &strategy.Sets[len(strategy.Sets)-1]
		lastSet.ParityShards += remainingDisks
		lastSet.TotalShards += remainingDisks
		lastSet.Disks = append(lastSet.Disks, makeRange(currentIndex, currentIndex+remainingDisks, diskMap)...)
	}

	return strategy
}

// calculateOptimalSetSize determines the best set size for a given number of disks
// Optimal set size is 16 shards (12 data + 4 parity) for larger deployments
func calculateOptimalSetSize(remainingDisks int32) int32 {
	if remainingDisks >= 32 {
		return 16 // Sweet spot for performance and reliability
	} else if remainingDisks >= 24 {
		return 16
	} else if remainingDisks >= 16 {
		return 16
	} else if remainingDisks >= 12 {
		return 12
	} else if remainingDisks >= 8 {
		return 8
	}
	return remainingDisks
}

// calculateShardDistribution calculates optimal data/parity shard distribution
// Aims for ~25-30% parity ratio
func calculateShardDistribution(totalShards int32) (dataShards, parityShards int32) {
	// Calculate parity shards (25-30% of total)
	parityShards = int32(math.Ceil(float64(totalShards) * 0.25))

	// Ensure minimum 2 parity shards
	if parityShards < 2 {
		parityShards = 2
	}

	// Ensure we don't exceed reasonable limits
	if parityShards > totalShards/2 {
		parityShards = totalShards / 3
		if parityShards < 2 {
			parityShards = 2
		}
	}

	dataShards = totalShards - parityShards
	return dataShards, parityShards
}

// makeRange creates a slice of integers from start (inclusive) to end (exclusive)
func makeRange(start, end int32, diskMap map[int]disk.Disk) []disk.Disk {
	result := make([]disk.Disk, end-start)
	s := int(start)
	for i := range result {
		result[i] = diskMap[s+i]
	}
	return result
}
