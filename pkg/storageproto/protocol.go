// Package storageproto описывает протокол HTTP-взаимодействия REST-сервисов хранения.
package storageproto

// Параметры REST-протокола взаимодействия со стораджами.
const (
	PartsPathFormat  = "%s/parts/%s/%d"
	HeaderChecksum   = "X-Checksum-Sha256"
	HeaderTotalParts = "X-Total-Parts"
	HeaderPartSize   = "X-Size"
)
