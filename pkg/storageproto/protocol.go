package storageproto

const (
	PartsPathFormat  = "%s/parts/%s/%d"
	HeaderChecksum   = "X-Checksum-Sha256"
	HeaderTotalParts = "X-Total-Parts"
	HeaderPartSize   = "X-Size"
)
