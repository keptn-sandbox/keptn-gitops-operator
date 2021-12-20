package utils

import (
	"github.com/mitchellh/hashstructure/v2"
	"strconv"
)

// ContainsString returns true if a string contains a substring
func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// GetHashStructure returns a hash value of the given struct
func GetHashStructure(i interface{}) string {
	hash, _ := hashstructure.Hash(i, hashstructure.FormatV2, nil)
	return strconv.FormatUint(hash, 10)
}
