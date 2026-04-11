package main

import (
	"testing"
)

func TestParseTagSingle(t *testing.T) {
	tags := ParseTag(`json:"name"`)
	if tags["json"] != "name" {
		t.Errorf("expected json=name, got %q", tags["json"])
	}
}

func TestParseTagMultiple(t *testing.T) {
	tags := ParseTag(`json:"port" yaml:"port" db:"server_port"`)
	if tags["json"] != "port" {
		t.Errorf("expected json=port, got %q", tags["json"])
	}
	if tags["yaml"] != "port" {
		t.Errorf("expected yaml=port, got %q", tags["yaml"])
	}
	if tags["db"] != "server_port" {
		t.Errorf("expected db=server_port, got %q", tags["db"])
	}
}

func TestParseTagWithOptions(t *testing.T) {
	tags := ParseTag(`json:"name,omitempty"`)
	if tags["json"] != "name,omitempty" {
		t.Errorf("expected json=name,omitempty, got %q", tags["json"])
	}
}

func TestSetTagKeyNew(t *testing.T) {
	result := SetTagKey(`json:"name"`, "yaml", "name")
	tags := ParseTag(result)
	if tags["json"] != "name" {
		t.Errorf("expected json=name preserved, got %q", tags["json"])
	}
	if tags["yaml"] != "name" {
		t.Errorf("expected yaml=name added, got %q", tags["yaml"])
	}
}

func TestSetTagKeyUpdate(t *testing.T) {
	result := SetTagKey(`json:"old_name" yaml:"old"`, "json", "new_name")
	tags := ParseTag(result)
	if tags["json"] != "new_name" {
		t.Errorf("expected json=new_name, got %q", tags["json"])
	}
	if tags["yaml"] != "old" {
		t.Errorf("expected yaml=old preserved, got %q", tags["yaml"])
	}
}

func TestSetTagKeyPreservesOrder(t *testing.T) {
	result := SetTagKey(`json:"a" yaml:"b" db:"c"`, "yaml", "updated")
	expected := `json:"a" yaml:"updated" db:"c"`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestRemoveTagKeyOne(t *testing.T) {
	result := RemoveTagKey(`json:"name" yaml:"name"`, "yaml")
	tags := ParseTag(result)
	if tags["json"] != "name" {
		t.Errorf("expected json=name preserved, got %q", tags["json"])
	}
	if _, ok := tags["yaml"]; ok {
		t.Error("expected yaml to be removed")
	}
}

func TestRemoveTagKeyAll(t *testing.T) {
	result := RemoveTagKey(`json:"name"`, "")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestRemoveTagKeyOnlyKey(t *testing.T) {
	result := RemoveTagKey(`json:"name"`, "json")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestSetTagKeyOnEmpty(t *testing.T) {
	result := SetTagKey("", "json", "name")
	expected := `json:"name"`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
