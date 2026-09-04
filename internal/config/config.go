// Package config defines the on-disk schema for a TUI and loads it into typed
// structs. The engine never sees the wire format: swapping in a second decoder
// (YAML, TOML) later only touches Parse.
package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Config is a complete TUI definition — one file, one interface.
type Config struct {
	Name        string            `json:"name"`
	Version     string            `json:"version,omitempty"`
	Keybindings map[string]string `json:"keybindings,omitempty"`
	Layout      Node              `json:"layout"`
}

// Node is a layout tree node: either a split (Direction + Children) or a leaf
// (Pane). Exactly one form must be populated. Presence, not a discriminator
// field, decides which — keeps hand-written configs terse.
type Node struct {
	// Direction is "row" (children left-to-right) or "column" (top-to-bottom).
	// Empty defaults to "column". Only meaningful for splits.
	Direction string  `json:"direction,omitempty"`
	Children  []Child `json:"children,omitempty"`
	Pane      *Pane   `json:"pane,omitempty"`
}

// Child is one slot in a split: a sizing hint plus a nested node.
type Child struct {
	// Size is "40%" (fraction of the split axis), "20" (fixed cells),
	// "2*" (flex weight), or "*" / "" (flex weight 1). Flex slots share
	// whatever space the fixed and percentage slots leave.
	Size string `json:"size,omitempty"`
	Node
}

// Pane is a leaf: a single widget plus chrome. Widget-specific keys live
// flat here for the MVP; a widget factory reads what it needs and ignores
// the rest.
type Pane struct {
	Type   string `json:"type"`             // registered widget kind
	Title  string `json:"title,omitempty"`  // shown as the pane header
	Border *bool  `json:"border,omitempty"` // nil == true

	// text widget
	Content string `json:"content,omitempty"`

	// command widget — runs a non-interactive command once and streams its
	// combined stdout/stderr into a scrollable pane. This is NOT an
	// interactive terminal (no pty, no vim/htop); that is a separate feature.
	Command string `json:"command,omitempty"`
	Shell   string `json:"shell,omitempty"` // default: $SHELL, else "sh"
}

// Load reads and parses a config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	c, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}

// Parse decodes config bytes (strict JSON — unknown keys are rejected so typos
// surface) and validates the result.
func Parse(data []byte) (*Config, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var c Config
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// Validate checks structural invariants that JSON typing cannot.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("config: \"name\" is required")
	}
	return validateNode(&c.Layout, "layout")
}

func validateNode(n *Node, path string) error {
	isSplit := len(n.Children) > 0
	isLeaf := n.Pane != nil
	switch {
	case isSplit && isLeaf:
		return fmt.Errorf("%s: node has both \"children\" and \"pane\"; pick one", path)
	case !isSplit && !isLeaf:
		return fmt.Errorf("%s: node has neither \"children\" nor \"pane\"", path)
	case isLeaf:
		if strings.TrimSpace(n.Pane.Type) == "" {
			return fmt.Errorf("%s.pane: \"type\" is required", path)
		}
		return nil
	default:
		if n.Direction != "" && n.Direction != "row" && n.Direction != "column" {
			return fmt.Errorf("%s: direction %q is not \"row\" or \"column\"", path, n.Direction)
		}
		for i := range n.Children {
			if err := validateNode(&n.Children[i].Node, fmt.Sprintf("%s.children[%d]", path, i)); err != nil {
				return err
			}
		}
		return nil
	}
}

// Leaves returns every pane in tree order — the focus ring.
func (c *Config) Leaves() []*Pane {
	var out []*Pane
	var walk func(n *Node)
	walk = func(n *Node) {
		if n.Pane != nil {
			out = append(out, n.Pane)
			return
		}
		for i := range n.Children {
			walk(&n.Children[i].Node)
		}
	}
	walk(&c.Layout)
	return out
}
