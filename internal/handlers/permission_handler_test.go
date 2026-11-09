package handlers

import (
	"testing"
)

func TestPermissionDefinitionsAndDefaults(t *testing.T) {
	defs := GetAllPermissionDefinitions()
	if len(defs) == 0 {
		t.Fatal("Expected non-empty permission definitions")
	}

	keys := GetAllPermissionKeys()
	if len(keys) == 0 {
		t.Fatal("Expected non-empty permission keys")
	}

	keyMap := make(map[string]bool)
	for _, k := range keys {
		if keyMap[k] {
			t.Fatalf("Duplicate permission key found: %s", k)
		}
		keyMap[k] = true
	}

	// Owner check: must have all keys
	ownerPerms := GetDefaultPermissionsForRole("owner")
	if len(ownerPerms) != len(keys) {
		t.Errorf("Expected owner to have all %d permissions, got %d", len(keys), len(ownerPerms))
	}

	// Admin check: should have most permissions, but not agenda.delete by default
	adminPerms := GetDefaultPermissionsForRole("admin")
	hasAdminDelete := false
	for _, p := range adminPerms {
		if p == "agenda.delete" {
			hasAdminDelete = true
		}
	}
	if hasAdminDelete {
		t.Error("Default admin permissions should not contain 'agenda.delete'")
	}

	// Crew check: should have agenda.view, picklist.checklist, inventory.view, but not financial
	crewPerms := GetDefaultPermissionsForRole("crew")
	crewMap := make(map[string]bool)
	for _, p := range crewPerms {
		crewMap[p] = true
	}
	if !crewMap["agenda.view"] || !crewMap["picklist.checklist"] || !crewMap["inventory.view"] {
		t.Error("Crew should have agenda.view, picklist.checklist, and inventory.view by default")
	}
	if crewMap["finance.view_summary"] || crewMap["finance.view_price"] || crewMap["agenda.create"] {
		t.Error("Crew should not have financial or agenda.create permissions by default")
	}
}
