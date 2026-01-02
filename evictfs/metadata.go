package evictfs

import (
	"github.com/gwangyi/fsx/contextual"
)

type lruMetadata struct {
	contextual.FileInfo
}

func newLRU(fi contextual.FileInfo) Metadata {
	return &lruMetadata{FileInfo: fi}
}

func (m *lruMetadata) Less(other Metadata) bool {
	otherFi := other.(*lruMetadata).FileInfo
	return m.FileInfo.AccessTime().Before(otherFi.AccessTime())
}

func (m *lruMetadata) Update(info contextual.FileInfo) {
	m.FileInfo = info
}
