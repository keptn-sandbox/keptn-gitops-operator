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

//CheckResponseCode checks if the response code meets an expected one
func CheckResponseCode(response *nethttp.Response, expectedCode int) error {
	if response.StatusCode != expectedCode {
		if response.Body == nil {
			return fmt.Errorf("could not read response body")
		}
		responseBody, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("could not read response body: %v", err)
		}
		return fmt.Errorf("%v", string(responseBody))
	}
	return nil
}
