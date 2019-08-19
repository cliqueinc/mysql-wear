package util

import (
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/pborman/uuid"
)

/*
IMPORTANT NOTE: guids created sequentially will LOOK THE SAME. Don't be fooled, they are in fact different

Customized GUID based on UUID1
We override the 48-36 bits of the UUID1 node value (MAC Hardware Addr) with our shard ID
Shard values are 0-4095 (12 bits)

This makes our GUIDs mostly incremental (time based) and sharded for scalability.
Also uses base32 to avoid annoying case sensitivity issues (IE)
See my original Python implementation:
https://gist.github.com/wrunk/b6d340297e7a9f3d97a0
*/

func GetShardId(guid string) (uint64, error) {

	idBytes, err := base32.StdEncoding.DecodeString(strings.ToUpper(guid) + "======")
	if err != nil {
		return 0, err
	}
	if len(idBytes) != 16 {
		msg := fmt.Sprintf("Decoded guid wasn't len 16! %v", len(idBytes))
		return 0, errors.New(msg)
	}
	i := binary.BigEndian.Uint64(append([]byte{0, 0}, idBytes[10:]...)) >> 36
	return i, nil
}

func NewGuidFromShard(affinityId string) string {
	// Return a new guid from the same shard as affinityId

	shardId, err := GetShardId(affinityId)
	if err != nil {
		// This would be very bad to not panic here. A bad or duplicated ID could end up
		// in the DB etc
		panic(err)
	}
	return _newGuid(shardId)
}

func NewGuid() string {
	return _newGuid(_getNewShardId())
}

func _getNewShardId() uint64 {
	return uint64(RandomInt(0, 4095))
}

func _newGuid(shardId uint64) string {
	/*
	   Customized GUID based on UUID1
	   We override the 48-36 bits of the UUID1 node value (MAC Hardware Addr) with our shard ID
	   Shard values are 0-4095 (12 bits)

	   This makes our GUIDs mostly incremental (time based) and sharded for scalability.
	   Also uses base32 to avoid annoying case sensitivity issues (IE)
	   See my original py implementation:
	   https://gist.github.com/wrunk/b6d340297e7a9f3d97a0
	*/
	if shardId > 4095 {
		msg := "shardId was too large"
		panic(msg)
	}

	// When we apply this mask with & we will free up the first 12 bits
	mask := uint64(0x000fffffffff)

	// Get the UUID1 node (mac address)
	nodeBytes := uuid.NodeID()

	// Get an 8 byte version of the node ID (MAC HW address) (otherwise Uint64 freaks out
	nodeBuf := append([]byte{0, 0}, nodeBytes...)

	// Get a uint64 version of the node ID (MAC HW address) to hold our 48bits
	nodeInt := binary.BigEndian.Uint64(nodeBuf)

	// Remove the left 12 bits
	maskedNode := nodeInt & mask

	// Shift our new shard id into place (left most 12 bits)
	shiftedShardId := shardId << 36

	finalNodeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(finalNodeBytes, shiftedShardId|maskedNode)

	// Finally set the UUID node
	uuid.SetNodeID(finalNodeBytes[2:])

	// This will return a UUID of base type []byte but has String() method implemented
	// which can cause confusion. Force it to be a []byte
	uid1 := []byte(uuid.NewUUID())

	// Since the uuid is of fixed len 32 we can remove the padding and re-add later
	return strings.TrimRight(strings.ToLower(base32.StdEncoding.EncodeToString(uid1)), "=")
}
