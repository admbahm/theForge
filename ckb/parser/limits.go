package parser

// Limits defines the security limits enforced during CKB parsing.
type Limits struct {
	MaxFileSize          int64
	MaxLineLength        int
	MaxMetadataRows      int
	MaxObjectCount       int
	MaxRelationshipsNode int
	MaxSectionCount      int
	MaxHeadingDepth      int
}

// DefaultLimits returns the standard production security limits.
func DefaultLimits() Limits {
	return Limits{
		MaxFileSize:          1024 * 1024, // 1MB
		MaxLineLength:        10000,       // 10,000 bytes
		MaxMetadataRows:      50,
		MaxObjectCount:       1000,
		MaxRelationshipsNode: 100,
		MaxSectionCount:      50,
		MaxHeadingDepth:      6,
	}
}
