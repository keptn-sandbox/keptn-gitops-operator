package utils

import (
	"github.com/mitchellh/hashstructure/v2"
	"strconv"
)

func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func GetHashStructure(i interface{}) string {
	hash, _ := hashstructure.Hash(i, hashstructure.FormatV2, nil)
	return strconv.FormatUint(hash, 10)
}
