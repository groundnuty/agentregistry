package models

import (
	"encoding/json"
	"fmt"
)

// ValidateA2AAgentCard validates raw JSON contains required Agent Card fields
// per the A2A v0.3 specification. Accepts both camelCase (Go SDK) and
// snake_case (Python SDK) field naming conventions.
//
// Required fields (reject if missing): name, description, url, version,
// capabilities, skills (non-empty), defaultInputModes, defaultOutputModes.
// Required per-skill fields: id, name, description.
// Recommended fields (warn if missing): protocolVersion, skill tags.
//
// protocolVersion is marked required by the official A2A Go SDK but is
// omitted from agentregistry's own arctl agent init template. We warn
// rather than reject to avoid breaking cards from the shipped tooling.
//
// The url field matches the Go SDK convention. The proto spec uses
// supported_interfaces instead, but all ecosystem SDKs use url.
func ValidateA2AAgentCard(raw json.RawMessage) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Required string fields
	for _, field := range []string{"name", "description", "url", "version"} {
		if !hasField(m, field) {
			return fmt.Errorf("required field %q is missing", field)
		}
	}

	// Required object
	if !hasField(m, "capabilities") {
		return fmt.Errorf("required field %q is missing", "capabilities")
	}

	// Required arrays (accept both naming conventions)
	requiredArrays := [][]string{
		{"skills"},
		{"defaultInputModes", "default_input_modes"},
		{"defaultOutputModes", "default_output_modes"},
	}
	for _, aliases := range requiredArrays {
		if !hasAnyField(m, aliases...) {
			return fmt.Errorf("required field %q is missing", aliases[0])
		}
	}

	// Validate skills array is non-empty and each skill has required fields
	if err := validateSkills(m["skills"]); err != nil {
		return err
	}

	return nil
}

func validateSkills(skillsRaw json.RawMessage) error {
	var skills []map[string]json.RawMessage
	if err := json.Unmarshal(skillsRaw, &skills); err != nil {
		return fmt.Errorf("skills must be a JSON array: %w", err)
	}
	if len(skills) == 0 {
		return fmt.Errorf("skills array must not be empty")
	}
	for i, skill := range skills {
		for _, field := range []string{"id", "name", "description"} {
			if _, ok := skill[field]; !ok {
				return fmt.Errorf("skill[%d] missing required field %q", i, field)
			}
		}
	}
	return nil
}

func hasField(m map[string]json.RawMessage, key string) bool {
	_, ok := m[key]
	return ok
}

func hasAnyField(m map[string]json.RawMessage, keys ...string) bool {
	for _, k := range keys {
		if _, ok := m[k]; ok {
			return true
		}
	}
	return false
}
