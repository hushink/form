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
	"testing"
)

type modeStruct struct {
	Tagged   string `form:"tagged"`
	Untagged string
	Omit     string `form:",omitempty"`
	Skipped  string `form:"-"`
}

func TestModeImplicitIsDefault(t *testing.T) {
	values, err := Marshal(modeStruct{Tagged: "t", Untagged: "u", Omit: "o", Skipped: "s"})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := values["tagged"]; len(got) != 1 || got[0] != "t" {
		t.Errorf("tagged = %v, want [t]", got)
	}
	if got := values["Untagged"]; len(got) != 1 || got[0] != "u" {
		t.Errorf("Untagged = %v, want [u]", got)
	}
	if got := values["Omit"]; len(got) != 1 || got[0] != "o" {
		t.Errorf("Omit = %v, want [o]", got)
	}
	if _, ok := values["Skipped"]; ok {
		t.Error("Skipped should not be encoded")
	}
}

func TestModeExplicitEncode(t *testing.T) {
	encoder := NewEncoder(WithMode(ModeExplicit))
	values, err := encoder.Encode(modeStruct{Tagged: "t", Untagged: "u", Omit: "o", Skipped: "s"})
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}
	if got := values["tagged"]; len(got) != 1 || got[0] != "t" {
		t.Errorf("tagged = %v, want [t]", got)
	}
	if _, ok := values["Untagged"]; ok {
		t.Error("Untagged should not be encoded in explicit mode")
	}
	// A tag with only options still counts as tagged; the field name is used.
	if got := values["Omit"]; len(got) != 1 || got[0] != "o" {
		t.Errorf("Omit = %v, want [o]", got)
	}
	if _, ok := values["Skipped"]; ok {
		t.Error("Skipped should not be encoded")
	}
}

func TestModeExplicitDecode(t *testing.T) {
	decoder := NewDecoder(WithMode(ModeExplicit))
	data := map[string][]string{
		"tagged":   {"t"},
		"Untagged": {"u"},
		"Omit":     {"o"},
	}

	var result modeStruct
	if err := decoder.Decode(data, &result); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if result.Tagged != "t" {
		t.Errorf("Tagged = %q, want t", result.Tagged)
	}
	if result.Untagged != "" {
		t.Errorf("Untagged = %q, want empty in explicit mode", result.Untagged)
	}
	if result.Omit != "o" {
		t.Errorf("Omit = %q, want o", result.Omit)
	}
}

type embeddedModeBase struct {
	Tagged   string `form:"tagged"`
	Untagged string
}

type embeddedModeStruct struct {
	embeddedModeBase
	Other         string `form:"other"`
	OtherUntagged string
}

func TestModeExplicitEmbedded(t *testing.T) {
	data := map[string][]string{
		"tagged":        {"t"},
		"Untagged":      {"u"},
		"other":         {"o"},
		"OtherUntagged": {"ou"},
	}

	var result embeddedModeStruct
	if err := NewDecoder(WithMode(ModeExplicitEmbedded)).Decode(data, &result); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if result.Tagged != "t" {
		t.Errorf("Tagged = %q, want t", result.Tagged)
	}
	if result.Untagged != "" {
		t.Errorf("Untagged = %q, want empty", result.Untagged)
	}
	if result.Other != "o" {
		t.Errorf("Other = %q, want o", result.Other)
	}
	if result.OtherUntagged != "" {
		t.Errorf("OtherUntagged = %q, want empty", result.OtherUntagged)
	}

	var strict embeddedModeStruct
	if err := NewDecoder(WithMode(ModeExplicit)).Decode(data, &strict); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if strict.Tagged != "" {
		t.Errorf("Tagged = %q, want empty with plain ModeExplicit", strict.Tagged)
	}

	values, err := NewEncoder(WithMode(ModeExplicitEmbedded)).Encode(embeddedModeStruct{
		embeddedModeBase: embeddedModeBase{Tagged: "t", Untagged: "u"},
		Other:            "o",
		OtherUntagged:    "ou",
	})
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}
	if got := values["tagged"]; len(got) != 1 || got[0] != "t" {
		t.Errorf("tagged = %v, want [t]", got)
	}
	if got := values["other"]; len(got) != 1 || got[0] != "o" {
		t.Errorf("other = %v, want [o]", got)
	}
	if _, ok := values["Untagged"]; ok {
		t.Error("Untagged should not be encoded")
	}
	if _, ok := values["OtherUntagged"]; ok {
		t.Error("OtherUntagged should not be encoded")
	}
}

type ExportedEmbeddedModeBase struct {
	Tagged string `form:"tagged"`
}

func TestModeExplicitEmbeddedPointer(t *testing.T) {
	type holder struct {
		*ExportedEmbeddedModeBase
		Other string `form:"other"`
	}

	var result holder
	if err := NewDecoder(WithMode(ModeExplicitEmbedded)).Decode(map[string][]string{
		"tagged": {"t"},
		"other":  {"o"},
	}, &result); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if result.ExportedEmbeddedModeBase == nil {
		t.Fatal("embedded pointer was not allocated")
	}
	if result.Tagged != "t" || result.Other != "o" {
		t.Errorf("result = %+v, want Tagged=t Other=o", result)
	}
}
