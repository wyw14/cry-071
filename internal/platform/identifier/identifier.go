package identifier

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"
)

type Generator interface {
	New(prefix string) string
}

type Random struct{}

func (Random) New(prefix string) string {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(buffer)
}

type Sequence struct{ value atomic.Uint64 }

func (s *Sequence) New(prefix string) string {
	return fmt.Sprintf("%s-%06d", prefix, s.value.Add(1))
}
