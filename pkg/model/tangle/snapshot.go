package tangle

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/iotaledger/hive.go/bitmask"
	"github.com/iotaledger/hive.go/syncutils"
	"github.com/iotaledger/iota.go/trinary"

	"github.com/gohornet/hornet/pkg/model/milestone"
)

const (
	SnapshotMetadataSpentAddressesEnabled = 0
)

var (
	snapshot                             *SnapshotInfo
	mutex                                syncutils.RWMutex
	latestSeenMilestoneIndexFromSnapshot = milestone.Index(0)

	ErrParseSnapshotInfoFailed = errors.New("Parsing of snapshot info failed")
)

type SnapshotInfo struct {
	CoordinatorAddress trinary.Hash
	Hash               trinary.Hash
	SnapshotIndex      milestone.Index
	PruningIndex       milestone.Index
	Timestamp          int64
	Metadata           bitmask.BitMask
}

func loadSnapshotInfo() {
	info, err := readSnapshotInfoFromDatabase()
	if err != nil {
		panic(err)
	}
	snapshot = info
	if info != nil {
		println(fmt.Sprintf("SnapshotInfo: CooAddr: %v, PruningIndex: %d, SnapshotIndex: %d (%v) Timestamp: %v, SpentAddressesEnabled: %v", info.CoordinatorAddress, info.PruningIndex, info.SnapshotIndex, info.Hash, time.Unix(info.Timestamp, 0).Truncate(time.Second), info.IsSpentAddressesEnabled()))
	}
}

func SnapshotInfoFromBytes(bytes []byte) (*SnapshotInfo, error) {

	if len(bytes) != 115 {
		return nil, errors.Wrapf(ErrParseSnapshotInfoFailed, "Invalid length %d != 115", len(bytes))
	}

	cooAddr := trinary.MustBytesToTrytes(bytes[:49], 81)
	hash := trinary.MustBytesToTrytes(bytes[49:98], 81)
	snapshotIndex := milestone.Index(binary.LittleEndian.Uint32(bytes[98:102]))
	pruningIndex := milestone.Index(binary.LittleEndian.Uint32(bytes[102:106]))
	timestamp := int64(binary.LittleEndian.Uint64(bytes[106:114]))
	metadata := bitmask.BitMask(bytes[114])

	return &SnapshotInfo{
		CoordinatorAddress: cooAddr,
		Hash:               hash,
		SnapshotIndex:      snapshotIndex,
		PruningIndex:       pruningIndex,
		Timestamp:          timestamp,
		Metadata:           metadata,
	}, nil
}

func (i *SnapshotInfo) IsSpentAddressesEnabled() bool {
	return i.Metadata.HasFlag(SnapshotMetadataSpentAddressesEnabled)
}

func (i *SnapshotInfo) SetSpentAddressesEnabled(enabled bool) {
	if enabled != i.Metadata.HasFlag(SnapshotMetadataSpentAddressesEnabled) {
		i.Metadata = i.Metadata.ModifyFlag(SnapshotMetadataSpentAddressesEnabled, enabled)
	}
}

func (i *SnapshotInfo) GetBytes() []byte {
	bytes := trinary.MustTrytesToBytes(i.CoordinatorAddress)[:49]

	bytes = append(bytes, trinary.MustTrytesToBytes(i.Hash)[:49]...)

	snapshotIndexBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(snapshotIndexBytes, uint32(i.SnapshotIndex))
	bytes = append(bytes, snapshotIndexBytes...)

	pruningIndexBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(pruningIndexBytes, uint32(i.PruningIndex))
	bytes = append(bytes, pruningIndexBytes...)

	timestampBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(timestampBytes, uint64(i.Timestamp))
	bytes = append(bytes, timestampBytes...)

	bytes = append(bytes, byte(i.Metadata))

	return bytes
}

func SetSnapshotMilestone(coordinatorAddress trinary.Hash, milestoneHash trinary.Hash, snapshotIndex milestone.Index, pruningIndex milestone.Index, timestamp int64, spentAddressesEnabled bool) {
	println(fmt.Sprintf("Loaded solid milestone from snapshot %d (%v), coo address: %v,  pruning index: %d, Timestamp: %v, SpentAddressesEnabled: %v", snapshotIndex, milestoneHash, coordinatorAddress, pruningIndex, time.Unix(timestamp, 0).Truncate(time.Second), spentAddressesEnabled))

	sn := &SnapshotInfo{
		CoordinatorAddress: coordinatorAddress,
		Hash:               milestoneHash,
		SnapshotIndex:      snapshotIndex,
		PruningIndex:       pruningIndex,
		Timestamp:          timestamp,
		Metadata:           bitmask.BitMask(0),
	}
	sn.SetSpentAddressesEnabled(spentAddressesEnabled)

	SetSnapshotInfo(sn)
}

func SetSnapshotInfo(sn *SnapshotInfo) {
	mutex.Lock()
	defer mutex.Unlock()

	err := storeSnapshotInfoInDatabase(sn)
	if err != nil {
		panic(err)
	}
	snapshot = sn
}

func GetSnapshotInfo() *SnapshotInfo {
	mutex.RLock()
	defer mutex.RUnlock()

	return snapshot
}

func SetLatestSeenMilestoneIndexFromSnapshot(milestoneIndex milestone.Index) {
	if latestSeenMilestoneIndexFromSnapshot < milestoneIndex {
		latestSeenMilestoneIndexFromSnapshot = milestoneIndex
	}
}

func GetLatestSeenMilestoneIndexFromSnapshot() milestone.Index {
	return latestSeenMilestoneIndexFromSnapshot
}
