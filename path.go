// Copyright 2026 hushink
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package form

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// pathNode is a node in the parsed form-key tree.
//
// Keys use go-playground/form syntax:
//
//	Struct.Field
//	Items[0].Name
//	Filter[key]
//	NestedMap[key][key2]
//
// Field segments are separated by the configured namespace prefix; bracket
// segments are interpreted as slice/array indexes or map keys depending on the
// destination type.
type pathNode struct {
	values   []string
	fields   map[string]*pathNode
	brackets []*pathEdge

	// path is the canonical dotted path used in errors and unknown-key
	// reporting. matched is set when the decoder reaches this path.
	path    string
	matched bool
}

type pathEdge struct {
	raw   string
	num   int
	isNum bool
	node  *pathNode

	path     string
	consumed bool
}

func (n *pathNode) hasContent() bool {
	return n != nil && (len(n.values) > 0 || len(n.fields) > 0 || len(n.brackets) > 0)
}

func (n *pathNode) field(name string) *pathNode {
	if n == nil || n.fields == nil {
		return nil
	}
	return n.fields[name]
}

// buildPathTree parses all input keys into a tree. Keys are sorted first so
// that traversal and error reporting are deterministic.
func buildPathTree(data map[string][]string, prefix, suffix string) (*pathNode, DecodeErrors) {
	root := &pathNode{matched: true}

	var errs DecodeErrors
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		segments, err := parseKeyPath(key, prefix, suffix)
		if err != nil {
			if errs == nil {
				errs = make(DecodeErrors)
			}
			errs[key] = err
			continue
		}

		node := root
		for _, seg := range segments {
			if seg.isBracket {
				var edge *pathEdge
				for _, candidate := range node.brackets {
					if candidate.raw == seg.bracket {
						edge = candidate
						break
					}
				}
				if edge == nil {
					edgePath := bracketPath(node.path, seg.bracket)
					edge = &pathEdge{
						raw:   seg.bracket,
						num:   seg.num,
						isNum: seg.isNum,
						path:  edgePath,
						node:  &pathNode{path: edgePath},
					}
					node.brackets = append(node.brackets, edge)
				}
				node = edge.node
				continue
			}

			if node.fields == nil {
				node.fields = make(map[string]*pathNode)
			}
			child := node.fields[seg.field]
			if child == nil {
				child = &pathNode{path: displayJoin(node.path, seg.field)}
				node.fields[seg.field] = child
			}
			node = child
		}

		node.values = append(node.values, data[key]...)
	}

	return root, errs
}

type pathSegment struct {
	field     string
	bracket   string
	isBracket bool
	num       int
	isNum     bool
}

// parseKeyPath splits a form key using the configured namespace prefix and
// suffix. The suffix is stripped from nested field names, mirroring the
// encoder's namespace construction.
func parseKeyPath(key, prefix, suffix string) ([]pathSegment, error) {
	// The empty key addresses the root value itself. This is used when
	// encoding/decoding a non-struct target such as a scalar or a slice.
	if key == "" {
		return nil, nil
	}
	if prefix == "" {
		prefix = defaultNamespacePrefix
	}

	var segments []pathSegment
	sawSegment := false
	i := 0
	for {
		// A leading bracket addresses the root container directly, e.g.
		// "[0]" for a top-level slice or "[key]" for a top-level map.
		if i != 0 || key[0] != '[' {
			start := i
			for i < len(key) && key[i] != '[' && !strings.HasPrefix(key[i:], prefix) {
				i++
			}
			name := key[start:i]
			if name == "" {
				return nil, fmt.Errorf("form: invalid path %q", key)
			}
			if sawSegment && suffix != "" {
				name = strings.TrimSuffix(name, suffix)
				if name == "" {
					return nil, fmt.Errorf("form: invalid path %q", key)
				}
			}
			segments = append(segments, pathSegment{field: name})
			if len(segments) > defaultMaxPathDepth {
				return nil, fmt.Errorf("form: path %q exceeds maximum depth %d", key, defaultMaxPathDepth)
			}
			sawSegment = true
		}

		for i < len(key) && key[i] == '[' {
			close := strings.IndexByte(key[i+1:], ']')
			if close < 0 {
				return nil, fmt.Errorf("form: invalid path %q: missing ']'", key)
			}
			content := key[i+1 : i+1+close]
			if content == "" {
				return nil, fmt.Errorf("form: invalid path %q: empty index", key)
			}

			seg := pathSegment{bracket: content, isBracket: true}
			if isAllDigits(content) {
				n, err := strconv.Atoi(content)
				if err != nil {
					return nil, fmt.Errorf("form: invalid index %q in path %q", content, key)
				}
				seg.isNum = true
				seg.num = n
			}
			segments = append(segments, seg)
			if len(segments) > defaultMaxPathDepth {
				return nil, fmt.Errorf("form: path %q exceeds maximum depth %d", key, defaultMaxPathDepth)
			}
			i += 1 + close + 1
			sawSegment = true
		}

		if i == len(key) {
			break
		}
		if !strings.HasPrefix(key[i:], prefix) {
			return nil, fmt.Errorf("form: invalid path %q", key)
		}
		i += len(prefix)
		if i == len(key) {
			return nil, fmt.Errorf("form: invalid path %q: trailing separator", key)
		}
	}

	return segments, nil
}

func isAllDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// displayJoin builds the canonical dotted path used in error messages.
func displayJoin(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

func indexedPath(path string, index int) string {
	return path + "[" + strconv.Itoa(index) + "]"
}

func bracketPath(path, key string) string {
	return path + "[" + key + "]"
}
