package utils

import (
	"fmt"
	"github.com/mitchellh/hashstructure/v2"
	"io/ioutil"
	nethttp "net/http"
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

func CheckResponseCode(response *nethttp.Response, expectedCode int) error {
	if response.StatusCode != expectedCode {
		responseBody, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("could not read response body: %v", err)
		}
		return fmt.Errorf("%v", responseBody)
	}
	return nil
}
