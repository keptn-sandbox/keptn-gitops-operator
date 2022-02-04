package v1

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
)

///// v0.2.0 Shipyard Spec ///////

// Shipyard describes a shipyard specification according to Keptn spec 0.2.0
type Shipyard struct {
	ApiVersion string       `json:"apiVersion" yaml:"apiVersion"`
	Kind       string       `json:"kind" yaml:"kind"`
	Metadata   Metadata     `json:"metadata" yaml:"metadata"`
	Spec       ShipyardSpec `json:"spec" yaml:"spec"`
}

// Metadata contains meta-data of a resource
type Metadata struct {
	Name string `json:"name" yaml:"name"`
}

// ShipyardSpec consists of any number of stages
type ShipyardSpec struct {
	Stages []Stage `json:"stages" yaml:"stages"`
}

// Stage defines a stage by its name and list of task sequences
type Stage struct {
	Name      string     `json:"name" yaml:"name"`
	Sequences []Sequence `json:"sequences,omitempty" yaml:"sequences,omitempty"`
}

// DecodeShipyardYAML takes a shipyard string formatted as YAML and decodes it to
// Shipyard value
func DecodeShipyardYAML(shipyardYaml []byte) (*Shipyard, error) {
	shipyardDecoded := &Shipyard{}

	if err := yaml.Unmarshal(shipyardYaml, shipyardDecoded); err != nil {
		return nil, err
	}
	return shipyardDecoded, nil
}

func Decode(in, out interface{}) error {
	bytes, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, out)
}
