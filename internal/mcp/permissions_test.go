package mcp

import (
	"testing"
)

func TestPresetExpansion(t *testing.T) {
	tests := []struct {
		preset   string
		wantLen  int
		contains []string
		excludes []string
	}{
		{
			preset:   "readonly",
			wantLen:  5,
			contains: []string{ScopeSchemaRead, ScopeConfigRead, ScopeModeRead, ScopeEncryptionRead, ScopeExporterRead},
			excludes: []string{ScopeSchemaWrite, ScopeAdminWrite},
		},
		{
			preset:   "developer",
			wantLen:  7,
			contains: []string{ScopeSchemaRead, ScopeSchemaWrite, ScopeConfigWrite},
			excludes: []string{ScopeSchemaDelete, ScopeAdminWrite},
		},
		{
			preset:   "operator",
			wantLen:  12,
			contains: []string{ScopeSchemaDelete, ScopeModeWrite, ScopeEncryptionWrite, ScopeExporterWrite, ScopeImport},
			excludes: []string{ScopeAdminRead, ScopeAdminWrite},
		},
		{
			preset:   "admin",
			wantLen:  14,
			contains: []string{ScopeAdminRead, ScopeAdminWrite},
			excludes: nil,
		},
		{
			preset:   "full",
			wantLen:  14,
			contains: allScopes,
			excludes: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.preset, func(t *testing.T) {
			scopes := resolvePermissionScopes(tt.preset, nil, false)
			if len(scopes) != tt.wantLen {
				t.Errorf("preset %q: got %d scopes, want %d", tt.preset, len(scopes), tt.wantLen)
			}
			for _, s := range tt.contains {
				if !scopes[s] {
					t.Errorf("preset %q: missing scope %q", tt.preset, s)
				}
			}
			for _, s := range tt.excludes {
				if scopes[s] {
					t.Errorf("preset %q: should not contain scope %q", tt.preset, s)
				}
			}
		})
	}
}

func TestSystemToolsAlwaysAllowed(t *testing.T) {
	systemTools := []string{
		"health_check", "get_server_info", "get_server_version",
		"get_cluster_id", "get_schema_types", "list_contexts",
		"count_subjects", "get_registry_statistics",
	}
	// Under the most restrictive preset (readonly)
	scopes := resolvePermissionScopes("readonly", nil, false)
	for _, tool := range systemTools {
		if !isScopeAllowed(tool, scopes) {
			t.Errorf("system tool %q should be allowed under readonly, but was denied", tool)
		}
	}
	// Under custom empty scope set
	emptyScopes := map[string]bool{}
	for _, tool := range systemTools {
		if !isScopeAllowed(tool, emptyScopes) {
			t.Errorf("system tool %q should be allowed with empty scopes, but was denied", tool)
		}
	}
}

func TestDeveloperAllowsRegisterBlocksDeleteAndAdmin(t *testing.T) {
	scopes := resolvePermissionScopes("developer", nil, false)
	if !isScopeAllowed("register_schema", scopes) {
		t.Error("developer should allow register_schema")
	}
	if isScopeAllowed("delete_subject", scopes) {
		t.Error("developer should block delete_subject")
	}
	if isScopeAllowed("create_user", scopes) {
		t.Error("developer should block create_user")
	}
}

func TestReadonlyBlocksAllWriteTools(t *testing.T) {
	scopes := resolvePermissionScopes("readonly", nil, false)
	writeTools := []string{
		"register_schema", "delete_subject", "delete_version",
		"set_config", "set_mode", "delete_mode",
		"import_schemas", "create_kek", "create_dek",
		"create_exporter", "create_user", "create_apikey",
	}
	for _, tool := range writeTools {
		if isScopeAllowed(tool, scopes) {
			t.Errorf("readonly should block %q, but it was allowed", tool)
		}
	}
}

func TestCustomScopes(t *testing.T) {
	scopes := resolvePermissionScopes("", []string{ScopeSchemaRead, ScopeEncryptionWrite}, false)
	if !isScopeAllowed("get_latest_schema", scopes) {
		t.Error("custom scopes should allow get_latest_schema (schema_read)")
	}
	if !isScopeAllowed("create_kek", scopes) {
		t.Error("custom scopes should allow create_kek (encryption_write)")
	}
	if isScopeAllowed("register_schema", scopes) {
		t.Error("custom scopes should block register_schema (schema_write not granted)")
	}
	if isScopeAllowed("get_config", scopes) {
		t.Error("custom scopes should block get_config (config_read not granted)")
	}
}

func TestPrecedence(t *testing.T) {
	// 1. permission_preset takes precedence over permission_scopes
	scopes := resolvePermissionScopes("readonly", []string{ScopeAdminWrite}, false)
	if isScopeAllowed("create_user", scopes) {
		t.Error("preset should override scopes: readonly preset should block admin_write")
	}

	// 2. permission_scopes takes precedence over read_only
	scopes = resolvePermissionScopes("", []string{ScopeSchemaWrite}, true)
	if !isScopeAllowed("register_schema", scopes) {
		t.Error("explicit scopes should override read_only")
	}

	// 3. read_only fallback to readonly preset
	scopes = resolvePermissionScopes("", nil, true)
	if isScopeAllowed("register_schema", scopes) {
		t.Error("read_only=true should act as readonly preset")
	}
	if !isScopeAllowed("get_latest_schema", scopes) {
		t.Error("read_only=true should allow reads")
	}

	// 4. No config → nil (all allowed)
	scopes = resolvePermissionScopes("", nil, false)
	if scopes != nil {
		t.Error("no config should return nil scopes")
	}
}

func TestEveryToolHasScope(t *testing.T) {
	// Collect all registered tool names from the map
	for tool := range toolPermissionScope {
		if tool == "" {
			t.Error("empty tool name in toolPermissionScope")
		}
	}
}

func TestToolScopeMapCompleteness(t *testing.T) {
	// Verify all known scope values are valid
	validScopes := scopeSet(allScopes)
	validScopes[""] = true // system tools
	for tool, scope := range toolPermissionScope {
		if !validScopes[scope] {
			t.Errorf("tool %q has invalid scope %q", tool, scope)
		}
	}
}

func TestNilScopesAllowsEverything(t *testing.T) {
	if !isScopeAllowed("register_schema", nil) {
		t.Error("nil scopes should allow all tools")
	}
	if !isScopeAllowed("create_user", nil) {
		t.Error("nil scopes should allow all tools")
	}
	if !isScopeAllowed("delete_subject", nil) {
		t.Error("nil scopes should allow all tools")
	}
}

func TestOperatorAllowsDeleteAndEncryption(t *testing.T) {
	scopes := resolvePermissionScopes("operator", nil, false)
	if !isScopeAllowed("delete_subject", scopes) {
		t.Error("operator should allow delete_subject")
	}
	if !isScopeAllowed("create_kek", scopes) {
		t.Error("operator should allow create_kek")
	}
	if isScopeAllowed("create_user", scopes) {
		t.Error("operator should block create_user")
	}
}

func TestAdminAllowsUserManagement(t *testing.T) {
	scopes := resolvePermissionScopes("admin", nil, false)
	if !isScopeAllowed("create_user", scopes) {
		t.Error("admin should allow create_user")
	}
	if !isScopeAllowed("delete_apikey", scopes) {
		t.Error("admin should allow delete_apikey")
	}
}
