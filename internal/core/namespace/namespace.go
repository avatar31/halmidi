package namespace

import "github.com/google/uuid"

const (
	DEFAULT_NAMESPACE  = "default"
	INTERNAL_NAMESPACE = "internal"
)

var (
	seed                 = uuid.MustParse("7f75691a-bfad-449e-bead-b5cfc8317112")
	DefaultNameSpaceUUID = generateNameSpaceUUID(DEFAULT_NAMESPACE)
)

func generateNameSpaceUUID(namespace string) string {
	return uuid.NewSHA1(seed, []byte(namespace)).String()
}
