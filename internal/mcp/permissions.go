package mcp

// Permission scopes mirror the REST RBAC taxonomy from internal/auth/rbac.go.
const (
	ScopeSchemaRead      = "schema_read"
	ScopeSchemaWrite     = "schema_write"
	ScopeSchemaDelete    = "schema_delete"
	ScopeConfigRead      = "config_read"
	ScopeConfigWrite     = "config_write"
	ScopeModeRead        = "mode_read"
	ScopeModeWrite       = "mode_write"
	ScopeImport          = "import"
	ScopeEncryptionRead  = "encryption_read"
	ScopeEncryptionWrite = "encryption_write"
	ScopeExporterRead    = "exporter_read"
	ScopeExporterWrite   = "exporter_write"
	ScopeAdminRead       = "admin_read"
	ScopeAdminWrite      = "admin_write"
)

// allScopes lists every permission scope.
var allScopes = []string{
	ScopeSchemaRead, ScopeSchemaWrite, ScopeSchemaDelete,
	ScopeConfigRead, ScopeConfigWrite,
	ScopeModeRead, ScopeModeWrite,
	ScopeImport,
	ScopeEncryptionRead, ScopeEncryptionWrite,
	ScopeExporterRead, ScopeExporterWrite,
	ScopeAdminRead, ScopeAdminWrite,
}

// permissionPresets maps preset names to their included scopes.
var permissionPresets = map[string][]string{
	"readonly": {
		ScopeSchemaRead, ScopeConfigRead, ScopeModeRead,
		ScopeEncryptionRead, ScopeExporterRead,
	},
	"developer": {
		ScopeSchemaRead, ScopeConfigRead, ScopeModeRead,
		ScopeEncryptionRead, ScopeExporterRead,
		ScopeSchemaWrite, ScopeConfigWrite,
	},
	"operator": {
		ScopeSchemaRead, ScopeConfigRead, ScopeModeRead,
		ScopeEncryptionRead, ScopeExporterRead,
		ScopeSchemaWrite, ScopeConfigWrite,
		ScopeSchemaDelete, ScopeModeWrite,
		ScopeEncryptionWrite, ScopeExporterWrite, ScopeImport,
	},
	"admin": {
		ScopeSchemaRead, ScopeConfigRead, ScopeModeRead,
		ScopeEncryptionRead, ScopeExporterRead,
		ScopeSchemaWrite, ScopeConfigWrite,
		ScopeSchemaDelete, ScopeModeWrite,
		ScopeEncryptionWrite, ScopeExporterWrite, ScopeImport,
		ScopeAdminRead, ScopeAdminWrite,
	},
	"full": nil, // nil = all scopes
}

// toolPermissionScope maps each tool name to the scope required to use it.
// Tools with an empty string scope are system tools that are always allowed.
var toolPermissionScope = map[string]string{
	// System tools (always allowed)
	"health_check":            "",
	"get_server_info":         "",
	"get_server_version":      "",
	"get_cluster_id":          "",
	"get_schema_types":        "",
	"list_contexts":           "",
	"count_subjects":          "",
	"get_registry_statistics": "",

	// schema_read
	"get_schema_by_id":              ScopeSchemaRead,
	"get_raw_schema_by_id":          ScopeSchemaRead,
	"get_schema_version":            ScopeSchemaRead,
	"get_raw_schema_version":        ScopeSchemaRead,
	"get_latest_schema":             ScopeSchemaRead,
	"list_versions":                 ScopeSchemaRead,
	"get_subjects_for_schema":       ScopeSchemaRead,
	"get_versions_for_schema":       ScopeSchemaRead,
	"get_referenced_by":             ScopeSchemaRead,
	"lookup_schema":                 ScopeSchemaRead,
	"list_schemas":                  ScopeSchemaRead,
	"get_max_schema_id":             ScopeSchemaRead,
	"list_subjects":                 ScopeSchemaRead,
	"get_schemas_by_subject":        ScopeSchemaRead,
	"get_schema_history":            ScopeSchemaRead,
	"get_dependency_graph":          ScopeSchemaRead,
	"export_schema":                 ScopeSchemaRead,
	"export_subject":                ScopeSchemaRead,
	"count_versions":                ScopeSchemaRead,
	"search_schemas":                ScopeSchemaRead,
	"match_subjects":                ScopeSchemaRead,
	"format_schema":                 ScopeSchemaRead,
	"resolve_alias":                 ScopeSchemaRead,
	"get_subject_metadata":          ScopeSchemaRead,
	"validate_schema":               ScopeSchemaRead,
	"normalize_schema":              ScopeSchemaRead,
	"validate_subject_name":         ScopeSchemaRead,
	"check_compatibility":           ScopeSchemaRead,
	"find_schemas_by_field":         ScopeSchemaRead,
	"find_schemas_by_type":          ScopeSchemaRead,
	"find_similar_schemas":          ScopeSchemaRead,
	"score_schema_quality":          ScopeSchemaRead,
	"check_field_consistency":       ScopeSchemaRead,
	"get_schema_complexity":         ScopeSchemaRead,
	"detect_schema_patterns":        ScopeSchemaRead,
	"suggest_schema_evolution":      ScopeSchemaRead,
	"plan_migration_path":           ScopeSchemaRead,
	"check_compatibility_multi":     ScopeSchemaRead,
	"diff_schemas":                  ScopeSchemaRead,
	"compare_subjects":              ScopeSchemaRead,
	"suggest_compatible_change":     ScopeSchemaRead,
	"explain_compatibility_failure": ScopeSchemaRead,

	// schema_write
	"register_schema": ScopeSchemaWrite,

	// schema_delete
	"delete_subject": ScopeSchemaDelete,
	"delete_version": ScopeSchemaDelete,

	// config_read
	"get_config":               ScopeConfigRead,
	"get_config_full":          ScopeConfigRead,
	"get_subject_config_full":  ScopeConfigRead,
	"get_global_config_direct": ScopeConfigRead,

	// config_write
	"set_config":      ScopeConfigWrite,
	"set_config_full": ScopeConfigWrite,
	"delete_config":   ScopeConfigWrite,

	// mode_read
	"get_mode":         ScopeModeRead,
	"check_write_mode": ScopeModeRead,

	// mode_write
	"set_mode":    ScopeModeWrite,
	"delete_mode": ScopeModeWrite,

	// import
	"import_schemas": ScopeImport,

	// encryption_read
	"get_kek":           ScopeEncryptionRead,
	"list_keks":         ScopeEncryptionRead,
	"get_dek":           ScopeEncryptionRead,
	"list_deks":         ScopeEncryptionRead,
	"list_dek_versions": ScopeEncryptionRead,

	// encryption_write
	"create_kek":   ScopeEncryptionWrite,
	"update_kek":   ScopeEncryptionWrite,
	"delete_kek":   ScopeEncryptionWrite,
	"undelete_kek": ScopeEncryptionWrite,
	"test_kek":     ScopeEncryptionWrite,
	"create_dek":   ScopeEncryptionWrite,
	"delete_dek":   ScopeEncryptionWrite,
	"undelete_dek": ScopeEncryptionWrite,
	"rewrap_dek":   ScopeEncryptionWrite,

	// exporter_read
	"list_exporters":      ScopeExporterRead,
	"get_exporter":        ScopeExporterRead,
	"get_exporter_status": ScopeExporterRead,
	"get_exporter_config": ScopeExporterRead,

	// exporter_write
	"create_exporter":        ScopeExporterWrite,
	"update_exporter":        ScopeExporterWrite,
	"delete_exporter":        ScopeExporterWrite,
	"pause_exporter":         ScopeExporterWrite,
	"resume_exporter":        ScopeExporterWrite,
	"reset_exporter":         ScopeExporterWrite,
	"update_exporter_config": ScopeExporterWrite,

	// admin_read
	"list_users":           ScopeAdminRead,
	"get_user":             ScopeAdminRead,
	"get_user_by_username": ScopeAdminRead,
	"list_apikeys":         ScopeAdminRead,
	"get_apikey":           ScopeAdminRead,
	"list_roles":           ScopeAdminRead,

	// admin_write
	"create_user":     ScopeAdminWrite,
	"update_user":     ScopeAdminWrite,
	"delete_user":     ScopeAdminWrite,
	"change_password": ScopeAdminWrite,
	"create_apikey":   ScopeAdminWrite,
	"update_apikey":   ScopeAdminWrite,
	"delete_apikey":   ScopeAdminWrite,
	"revoke_apikey":   ScopeAdminWrite,
	"rotate_apikey":   ScopeAdminWrite,
}

// resolvePermissionScopes computes the effective set of allowed scopes based
// on the configuration. Resolution order:
//  1. permission_preset → expand to preset scopes
//  2. permission_scopes → use listed scopes
//  3. read_only → equivalent to "readonly" preset
//  4. Default → nil (all allowed, fall through to tool_policy)
func resolvePermissionScopes(preset string, scopes []string, readOnly bool) map[string]bool {
	// 1. Named preset
	if preset != "" {
		if presetScopes, ok := permissionPresets[preset]; ok {
			if presetScopes == nil {
				// "full" — all scopes
				return scopeSet(allScopes)
			}
			return scopeSet(presetScopes)
		}
	}

	// 2. Explicit scopes list
	if len(scopes) > 0 {
		return scopeSet(scopes)
	}

	// 3. read_only flag → readonly preset
	if readOnly {
		return scopeSet(permissionPresets["readonly"])
	}

	// 4. No permission config → nil means "not using permission scopes"
	return nil
}

// isScopeAllowed checks whether a tool is allowed given the resolved scopes.
// System tools (empty scope) are always allowed.
// A nil resolvedScopes means scopes are not configured — always allowed.
func isScopeAllowed(toolName string, resolvedScopes map[string]bool) bool {
	scope, exists := toolPermissionScope[toolName]
	if !exists {
		// Unknown tool — allow (will be caught by tool_policy or registration)
		return true
	}
	if scope == "" {
		// System tool — always allowed
		return true
	}
	if resolvedScopes == nil {
		// Scopes not configured — allow all
		return true
	}
	return resolvedScopes[scope]
}

func scopeSet(scopes []string) map[string]bool {
	m := make(map[string]bool, len(scopes))
	for _, s := range scopes {
		m[s] = true
	}
	return m
}
