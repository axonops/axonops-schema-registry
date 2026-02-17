// Package main is the entry point for the schema registry admin CLI.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/axonops/axonops-schema-registry/internal/auth"
	"github.com/axonops/axonops-schema-registry/internal/storage"
	"github.com/axonops/axonops-schema-registry/internal/storage/cassandra"
	"github.com/axonops/axonops-schema-registry/internal/storage/memory"
	"github.com/axonops/axonops-schema-registry/internal/storage/mysql"
	"github.com/axonops/axonops-schema-registry/internal/storage/postgres"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

var (
	serverURL string
	username  string
	password  string
	apiKey    string
	output    string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "schema-registry-admin",
		Short: "Admin CLI for AxonOps Schema Registry",
		Long:  `A command-line tool for managing users, API keys, and roles in the AxonOps Schema Registry.`,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&serverURL, "server", "s", "http://localhost:8081", "Schema Registry server URL")
	rootCmd.PersistentFlags().StringVarP(&username, "username", "u", "", "Username for basic auth")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "Password for basic auth")
	rootCmd.PersistentFlags().StringVarP(&apiKey, "api-key", "k", "", "API key for authentication")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "table", "Output format: table, json")

	// User commands
	userCmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
	}

	userListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all users",
		RunE:  listUsers,
	}

	userGetCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get user by ID",
		Args:  cobra.ExactArgs(1),
		RunE:  getUser,
	}

	userCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new user",
		RunE:  createUser,
	}
	userCreateCmd.Flags().String("name", "", "Username (required)")
	userCreateCmd.Flags().String("email", "", "Email address")
	userCreateCmd.Flags().String("pass", "", "Password (required)")
	userCreateCmd.Flags().String("role", "", "Role: super_admin, admin, developer, readonly (required)")
	userCreateCmd.Flags().Bool("enabled", true, "Whether the user is enabled")
	_ = userCreateCmd.MarkFlagRequired("name")
	_ = userCreateCmd.MarkFlagRequired("pass")
	_ = userCreateCmd.MarkFlagRequired("role")

	userUpdateCmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a user",
		Args:  cobra.ExactArgs(1),
		RunE:  updateUser,
	}
	userUpdateCmd.Flags().String("email", "", "Email address")
	userUpdateCmd.Flags().String("pass", "", "New password")
	userUpdateCmd.Flags().String("role", "", "Role: super_admin, admin, developer, readonly")
	userUpdateCmd.Flags().Bool("enabled", false, "Whether the user is enabled")
	userUpdateCmd.Flags().Bool("disabled", false, "Disable the user")

	userDeleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a user",
		Args:  cobra.ExactArgs(1),
		RunE:  deleteUser,
	}

	userCmd.AddCommand(userListCmd, userGetCmd, userCreateCmd, userUpdateCmd, userDeleteCmd)

	// API Key commands
	apikeyCmd := &cobra.Command{
		Use:     "apikey",
		Aliases: []string{"key"},
		Short:   "Manage API keys",
	}

	apikeyListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all API keys",
		RunE:  listAPIKeys,
	}
	apikeyListCmd.Flags().Int64("user-id", 0, "Filter by user ID")

	apikeyGetCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get API key by ID",
		Args:  cobra.ExactArgs(1),
		RunE:  getAPIKey,
	}

	apikeyCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new API key",
		RunE:  createAPIKey,
	}
	apikeyCreateCmd.Flags().String("name", "", "API key name, unique per user (required)")
	apikeyCreateCmd.Flags().String("role", "", "Role: super_admin, admin, developer, readonly (required)")
	apikeyCreateCmd.Flags().Duration("expires-in", 0, "Expiration duration (required, e.g., 720h for 30 days, 8760h for 1 year)")
	apikeyCreateCmd.Flags().Int64("for-user-id", 0, "Create API key for another user (super_admin only)")
	_ = apikeyCreateCmd.MarkFlagRequired("name")
	_ = apikeyCreateCmd.MarkFlagRequired("role")
	_ = apikeyCreateCmd.MarkFlagRequired("expires-in")

	apikeyUpdateCmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an API key",
		Args:  cobra.ExactArgs(1),
		RunE:  updateAPIKey,
	}
	apikeyUpdateCmd.Flags().String("name", "", "API key name")
	apikeyUpdateCmd.Flags().String("role", "", "Role: super_admin, admin, developer, readonly")
	apikeyUpdateCmd.Flags().Bool("enabled", false, "Enable the API key")
	apikeyUpdateCmd.Flags().Bool("disabled", false, "Disable the API key")

	apikeyDeleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an API key",
		Args:  cobra.ExactArgs(1),
		RunE:  deleteAPIKey,
	}

	apikeyRevokeCmd := &cobra.Command{
		Use:   "revoke <id>",
		Short: "Revoke (disable) an API key",
		Args:  cobra.ExactArgs(1),
		RunE:  revokeAPIKey,
	}

	apikeyRotateCmd := &cobra.Command{
		Use:   "rotate <id>",
		Short: "Rotate an API key (create new, revoke old)",
		Args:  cobra.ExactArgs(1),
		RunE:  rotateAPIKey,
	}
	apikeyRotateCmd.Flags().Duration("expires-in", 0, "Expiration duration for new key (required, e.g., 720h for 30 days)")
	_ = apikeyRotateCmd.MarkFlagRequired("expires-in")

	apikeyCmd.AddCommand(apikeyListCmd, apikeyGetCmd, apikeyCreateCmd, apikeyUpdateCmd, apikeyDeleteCmd, apikeyRevokeCmd, apikeyRotateCmd)

	// Role commands
	roleCmd := &cobra.Command{
		Use:   "role",
		Short: "Manage roles",
	}

	roleListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all available roles",
		RunE:  listRoles,
	}

	roleCmd.AddCommand(roleListCmd)

	// Version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("schema-registry-admin %s (commit: %s, built: %s)\n", version, commit, buildDate)
		},
	}

	// Init command - bootstrap initial admin user directly in database
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the schema registry with an admin user",
		Long: `Initialize the schema registry by creating an initial admin user directly in the database.

This command bypasses the API and connects directly to the database to create
the first super_admin user. Use this when you need to bootstrap a fresh deployment
where no users exist yet.

Examples:
  # Initialize with PostgreSQL
  schema-registry-admin init --storage-type postgresql \
    --pg-host localhost --pg-port 5432 --pg-database schema_registry \
    --pg-user postgres --pg-password secret \
    --admin-username admin --admin-password 'secure-password'

  # Initialize with MySQL
  schema-registry-admin init --storage-type mysql \
    --mysql-host localhost --mysql-port 3306 --mysql-database schema_registry \
    --mysql-user root --mysql-password secret \
    --admin-username admin --admin-password 'secure-password'

  # Initialize with Cassandra
  schema-registry-admin init --storage-type cassandra \
    --cassandra-hosts localhost --cassandra-keyspace schema_registry \
    --admin-username admin --admin-password 'secure-password'

Environment variables can also be used:
  SCHEMA_REGISTRY_PG_HOST, SCHEMA_REGISTRY_PG_PORT, etc.
  SCHEMA_REGISTRY_MYSQL_HOST, SCHEMA_REGISTRY_MYSQL_PORT, etc.
  SCHEMA_REGISTRY_CASSANDRA_HOSTS, etc.
`,
		RunE: initAdmin,
	}
	// Storage type
	initCmd.Flags().String("storage-type", "postgresql", "Storage type: postgresql, mysql, cassandra, memory")
	// PostgreSQL flags
	initCmd.Flags().String("pg-host", getEnvOrDefault("SCHEMA_REGISTRY_PG_HOST", "localhost"), "PostgreSQL host")
	initCmd.Flags().Int("pg-port", getEnvOrDefaultInt("SCHEMA_REGISTRY_PG_PORT", 5432), "PostgreSQL port")
	initCmd.Flags().String("pg-database", getEnvOrDefault("SCHEMA_REGISTRY_PG_DATABASE", "schema_registry"), "PostgreSQL database")
	initCmd.Flags().String("pg-user", getEnvOrDefault("SCHEMA_REGISTRY_PG_USER", ""), "PostgreSQL user")
	initCmd.Flags().String("pg-password", getEnvOrDefault("SCHEMA_REGISTRY_PG_PASSWORD", ""), "PostgreSQL password")
	initCmd.Flags().String("pg-sslmode", getEnvOrDefault("SCHEMA_REGISTRY_PG_SSLMODE", "disable"), "PostgreSQL SSL mode")
	// MySQL flags
	initCmd.Flags().String("mysql-host", getEnvOrDefault("SCHEMA_REGISTRY_MYSQL_HOST", "localhost"), "MySQL host")
	initCmd.Flags().Int("mysql-port", getEnvOrDefaultInt("SCHEMA_REGISTRY_MYSQL_PORT", 3306), "MySQL port")
	initCmd.Flags().String("mysql-database", getEnvOrDefault("SCHEMA_REGISTRY_MYSQL_DATABASE", "schema_registry"), "MySQL database")
	initCmd.Flags().String("mysql-user", getEnvOrDefault("SCHEMA_REGISTRY_MYSQL_USER", ""), "MySQL user")
	initCmd.Flags().String("mysql-password", getEnvOrDefault("SCHEMA_REGISTRY_MYSQL_PASSWORD", ""), "MySQL password")
	initCmd.Flags().String("mysql-tls", getEnvOrDefault("SCHEMA_REGISTRY_MYSQL_TLS", "false"), "MySQL TLS mode")
	// Cassandra flags
	initCmd.Flags().String("cassandra-hosts", getEnvOrDefault("SCHEMA_REGISTRY_CASSANDRA_HOSTS", "localhost"), "Cassandra hosts (comma-separated)")
	initCmd.Flags().String("cassandra-keyspace", getEnvOrDefault("SCHEMA_REGISTRY_CASSANDRA_KEYSPACE", "schema_registry"), "Cassandra keyspace")
	initCmd.Flags().String("cassandra-username", getEnvOrDefault("SCHEMA_REGISTRY_CASSANDRA_USERNAME", ""), "Cassandra username")
	initCmd.Flags().String("cassandra-password", getEnvOrDefault("SCHEMA_REGISTRY_CASSANDRA_PASSWORD", ""), "Cassandra password")
	initCmd.Flags().String("cassandra-consistency", getEnvOrDefault("SCHEMA_REGISTRY_CASSANDRA_CONSISTENCY", "LOCAL_QUORUM"), "Cassandra consistency")
	// Admin user flags
	initCmd.Flags().String("admin-username", getEnvOrDefault("SCHEMA_REGISTRY_BOOTSTRAP_USERNAME", "admin"), "Admin username")
	initCmd.Flags().String("admin-password", getEnvOrDefault("SCHEMA_REGISTRY_BOOTSTRAP_PASSWORD", ""), "Admin password (required)")
	initCmd.Flags().String("admin-email", getEnvOrDefault("SCHEMA_REGISTRY_BOOTSTRAP_EMAIL", ""), "Admin email (optional)")
	_ = initCmd.MarkFlagRequired("admin-password")

	rootCmd.AddCommand(userCmd, apikeyCmd, roleCmd, versionCmd, initCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// HTTP client helper
func doRequest(method, path string, body interface{}) (map[string]interface{}, error) {
	url := strings.TrimSuffix(serverURL, "/") + path

	var req *http.Request
	var err error

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		req, err = http.NewRequest(method, url, strings.NewReader(string(jsonBody)))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
	}

	// Authentication
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	} else if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req) // #nosec G704 -- admin CLI tool; URL is from user-provided --server flag
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil && resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode >= 400 {
		msg := "unknown error"
		if m, ok := result["message"].(string); ok {
			msg = m
		}
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, msg)
	}

	return result, nil
}

// User commands
func listUsers(cmd *cobra.Command, args []string) error {
	result, err := doRequest("GET", "/admin/users", nil)
	if err != nil {
		return err
	}

	users, ok := result["users"].([]interface{})
	if !ok {
		return fmt.Errorf("unexpected response format")
	}

	if output == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(users)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tUSERNAME\tEMAIL\tROLE\tENABLED\tCREATED")
	for _, u := range users {
		user := u.(map[string]interface{})
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\n",
			int64(user["id"].(float64)),
			user["username"],
			user["email"],
			user["role"],
			user["enabled"],
			formatTime(user["created_at"]),
		)
	}
	return w.Flush()
}

func getUser(cmd *cobra.Command, args []string) error {
	result, err := doRequest("GET", "/admin/users/"+args[0], nil)
	if err != nil {
		return err
	}

	if output == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	fmt.Printf("ID:       %v\n", int64(result["id"].(float64)))
	fmt.Printf("Username: %v\n", result["username"])
	fmt.Printf("Email:    %v\n", result["email"])
	fmt.Printf("Role:     %v\n", result["role"])
	fmt.Printf("Enabled:  %v\n", result["enabled"])
	fmt.Printf("Created:  %v\n", formatTime(result["created_at"]))
	fmt.Printf("Updated:  %v\n", formatTime(result["updated_at"]))
	return nil
}

func createUser(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	email, _ := cmd.Flags().GetString("email")
	pass, _ := cmd.Flags().GetString("pass")
	role, _ := cmd.Flags().GetString("role")
	enabled, _ := cmd.Flags().GetBool("enabled")

	body := map[string]interface{}{
		"username": name,
		"password": pass,
		"role":     role,
		"enabled":  enabled,
	}
	if email != "" {
		body["email"] = email
	}

	result, err := doRequest("POST", "/admin/users", body)
	if err != nil {
		return err
	}

	if output == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	fmt.Printf("User created successfully!\n")
	fmt.Printf("ID:       %v\n", int64(result["id"].(float64)))
	fmt.Printf("Username: %v\n", result["username"])
	fmt.Printf("Role:     %v\n", result["role"])
	return nil
}

func updateUser(cmd *cobra.Command, args []string) error {
	body := make(map[string]interface{})

	if cmd.Flags().Changed("email") {
		email, _ := cmd.Flags().GetString("email")
		body["email"] = email
	}
	if cmd.Flags().Changed("pass") {
		pass, _ := cmd.Flags().GetString("pass")
		body["password"] = pass
	}
	if cmd.Flags().Changed("role") {
		role, _ := cmd.Flags().GetString("role")
		body["role"] = role
	}
	if cmd.Flags().Changed("enabled") {
		body["enabled"] = true
	}
	if cmd.Flags().Changed("disabled") {
		body["enabled"] = false
	}

	if len(body) == 0 {
		return fmt.Errorf("no fields to update")
	}

	result, err := doRequest("PUT", "/admin/users/"+args[0], body)
	if err != nil {
		return err
	}

	if output == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	fmt.Printf("User updated successfully!\n")
	fmt.Printf("ID:       %v\n", int64(result["id"].(float64)))
	fmt.Printf("Username: %v\n", result["username"])
	fmt.Printf("Role:     %v\n", result["role"])
	fmt.Printf("Enabled:  %v\n", result["enabled"])
	return nil
}

func deleteUser(cmd *cobra.Command, args []string) error {
	_, err := doRequest("DELETE", "/admin/users/"+args[0], nil)
	if err != nil {
		return err
	}

	fmt.Println("User deleted successfully!")
	return nil
}

// API Key commands
func listAPIKeys(cmd *cobra.Command, args []string) error {
	path := "/admin/apikeys"
	userID, _ := cmd.Flags().GetInt64("user-id")
	if userID > 0 {
		path += "?user_id=" + strconv.FormatInt(userID, 10)
	}

	result, err := doRequest("GET", path, nil)
	if err != nil {
		return err
	}

	keys, ok := result["api_keys"].([]interface{})
	if !ok {
		return fmt.Errorf("unexpected response format")
	}

	if output == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(keys)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tPREFIX\tNAME\tROLE\tENABLED\tEXPIRES\tLAST USED")
	for _, k := range keys {
		key := k.(map[string]interface{})
		expires := "-"
		if e := key["expires_at"]; e != nil {
			expires = formatTime(e)
		}
		lastUsed := "-"
		if lu := key["last_used"]; lu != nil {
			lastUsed = formatTime(lu)
		}
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\n",
			int64(key["id"].(float64)),
			key["key_prefix"],
			key["name"],
			key["role"],
			key["enabled"],
			expires,
			lastUsed,
		)
	}
	return w.Flush()
}

func getAPIKey(cmd *cobra.Command, args []string) error {
	result, err := doRequest("GET", "/admin/apikeys/"+args[0], nil)
	if err != nil {
		return err
	}

	if output == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	fmt.Printf("ID:         %v\n", int64(result["id"].(float64)))
	fmt.Printf("Key Prefix: %v\n", result["key_prefix"])
	fmt.Printf("Name:       %v\n", result["name"])
	fmt.Printf("Role:       %v\n", result["role"])
	if userID := result["user_id"]; userID != nil {
		fmt.Printf("User ID:    %v\n", int64(userID.(float64)))
	}
	fmt.Printf("Enabled:    %v\n", result["enabled"])
	fmt.Printf("Created:    %v\n", formatTime(result["created_at"]))
	if expires := result["expires_at"]; expires != nil {
		fmt.Printf("Expires:    %v\n", formatTime(expires))
	}
	if lastUsed := result["last_used"]; lastUsed != nil {
		fmt.Printf("Last Used:  %v\n", formatTime(lastUsed))
	}
	return nil
}

func createAPIKey(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	role, _ := cmd.Flags().GetString("role")
	expiresIn, _ := cmd.Flags().GetDuration("expires-in")
	forUserID, _ := cmd.Flags().GetInt64("for-user-id")

	body := map[string]interface{}{
		"name":       name,
		"role":       role,
		"expires_in": int64(expiresIn.Seconds()),
	}
	if forUserID > 0 {
		body["for_user_id"] = forUserID
	}

	result, err := doRequest("POST", "/admin/apikeys", body)
	if err != nil {
		return err
	}

	if output == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	fmt.Printf("API Key created successfully!\n")
	fmt.Printf("\n")
	fmt.Printf("IMPORTANT: Save this key now - it won't be shown again!\n")
	fmt.Printf("\n")
	fmt.Printf("Key:        %v\n", result["key"])
	fmt.Printf("ID:         %v\n", int64(result["id"].(float64)))
	fmt.Printf("Key Prefix: %v\n", result["key_prefix"])
	fmt.Printf("Name:       %v\n", result["name"])
	fmt.Printf("Role:       %v\n", result["role"])
	fmt.Printf("Username:   %v\n", result["username"])
	fmt.Printf("Expires:    %v\n", formatTime(result["expires_at"]))
	return nil
}

func updateAPIKey(cmd *cobra.Command, args []string) error {
	body := make(map[string]interface{})

	if cmd.Flags().Changed("name") {
		name, _ := cmd.Flags().GetString("name")
		body["name"] = name
	}
	if cmd.Flags().Changed("role") {
		role, _ := cmd.Flags().GetString("role")
		body["role"] = role
	}
	if cmd.Flags().Changed("enabled") {
		body["enabled"] = true
	}
	if cmd.Flags().Changed("disabled") {
		body["enabled"] = false
	}

	if len(body) == 0 {
		return fmt.Errorf("no fields to update")
	}

	result, err := doRequest("PUT", "/admin/apikeys/"+args[0], body)
	if err != nil {
		return err
	}

	if output == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	fmt.Printf("API Key updated successfully!\n")
	fmt.Printf("ID:      %v\n", int64(result["id"].(float64)))
	fmt.Printf("Name:    %v\n", result["name"])
	fmt.Printf("Role:    %v\n", result["role"])
	fmt.Printf("Enabled: %v\n", result["enabled"])
	return nil
}

func deleteAPIKey(cmd *cobra.Command, args []string) error {
	_, err := doRequest("DELETE", "/admin/apikeys/"+args[0], nil)
	if err != nil {
		return err
	}

	fmt.Println("API Key deleted successfully!")
	return nil
}

func revokeAPIKey(cmd *cobra.Command, args []string) error {
	result, err := doRequest("POST", "/admin/apikeys/"+args[0]+"/revoke", nil)
	if err != nil {
		return err
	}

	if output == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	fmt.Printf("API Key revoked successfully!\n")
	fmt.Printf("ID:      %v\n", int64(result["id"].(float64)))
	fmt.Printf("Enabled: %v\n", result["enabled"])
	return nil
}

func rotateAPIKey(cmd *cobra.Command, args []string) error {
	expiresIn, _ := cmd.Flags().GetDuration("expires-in")

	body := map[string]interface{}{
		"expires_in": int64(expiresIn.Seconds()),
	}

	result, err := doRequest("POST", "/admin/apikeys/"+args[0]+"/rotate", body)
	if err != nil {
		return err
	}

	if output == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	newKey := result["new_key"].(map[string]interface{})

	fmt.Printf("API Key rotated successfully!\n")
	fmt.Printf("\n")
	fmt.Printf("IMPORTANT: Save this new key now - it won't be shown again!\n")
	fmt.Printf("\n")
	fmt.Printf("New Key:    %v\n", newKey["key"])
	fmt.Printf("New ID:     %v\n", int64(newKey["id"].(float64)))
	fmt.Printf("Revoked ID: %v\n", int64(result["revoked_id"].(float64)))
	return nil
}

// Role commands
func listRoles(cmd *cobra.Command, args []string) error {
	result, err := doRequest("GET", "/admin/roles", nil)
	if err != nil {
		return err
	}

	roles, ok := result["roles"].([]interface{})
	if !ok {
		return fmt.Errorf("unexpected response format")
	}

	if output == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(roles)
	}

	for _, r := range roles {
		role := r.(map[string]interface{})
		fmt.Printf("%v\n", role["name"])
		fmt.Printf("  Description: %v\n", role["description"])
		fmt.Printf("  Permissions:\n")
		perms := role["permissions"].([]interface{})
		for _, p := range perms {
			fmt.Printf("    - %v\n", p)
		}
		fmt.Println()
	}
	return nil
}

// Helpers
func formatTime(t interface{}) string {
	if t == nil {
		return "-"
	}
	s, ok := t.(string)
	if !ok {
		return fmt.Sprintf("%v", t)
	}
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return parsed.Local().Format("2006-01-02 15:04:05")
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvOrDefaultInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultValue
}

// initAdmin bootstraps the initial admin user directly in the database.
func initAdmin(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	storageType, _ := cmd.Flags().GetString("storage-type")
	adminUsername, _ := cmd.Flags().GetString("admin-username")
	adminPassword, _ := cmd.Flags().GetString("admin-password")
	adminEmail, _ := cmd.Flags().GetString("admin-email")

	fmt.Printf("Connecting to %s storage...\n", storageType)

	// Create storage backend
	var store interface {
		storage.AuthStorage
		Close() error
	}
	var err error

	switch storageType {
	case "postgresql", "postgres":
		pgHost, _ := cmd.Flags().GetString("pg-host")
		pgPort, _ := cmd.Flags().GetInt("pg-port")
		pgDatabase, _ := cmd.Flags().GetString("pg-database")
		pgUser, _ := cmd.Flags().GetString("pg-user")
		pgPassword, _ := cmd.Flags().GetString("pg-password")
		pgSSLMode, _ := cmd.Flags().GetString("pg-sslmode")

		cfg := postgres.Config{
			Host:         pgHost,
			Port:         pgPort,
			Database:     pgDatabase,
			Username:     pgUser,
			Password:     pgPassword,
			SSLMode:      pgSSLMode,
			MaxOpenConns: 5,
			MaxIdleConns: 2,
		}
		store, err = postgres.NewStore(cfg)

	case "mysql":
		mysqlHost, _ := cmd.Flags().GetString("mysql-host")
		mysqlPort, _ := cmd.Flags().GetInt("mysql-port")
		mysqlDatabase, _ := cmd.Flags().GetString("mysql-database")
		mysqlUser, _ := cmd.Flags().GetString("mysql-user")
		mysqlPassword, _ := cmd.Flags().GetString("mysql-password")
		mysqlTLS, _ := cmd.Flags().GetString("mysql-tls")

		cfg := mysql.Config{
			Host:         mysqlHost,
			Port:         mysqlPort,
			Database:     mysqlDatabase,
			Username:     mysqlUser,
			Password:     mysqlPassword,
			TLS:          mysqlTLS,
			MaxOpenConns: 5,
			MaxIdleConns: 2,
		}
		store, err = mysql.NewStore(cfg)

	case "cassandra":
		cassandraHosts, _ := cmd.Flags().GetString("cassandra-hosts")
		cassandraKeyspace, _ := cmd.Flags().GetString("cassandra-keyspace")
		cassandraUsername, _ := cmd.Flags().GetString("cassandra-username")
		cassandraPassword, _ := cmd.Flags().GetString("cassandra-password")
		cassandraConsistency, _ := cmd.Flags().GetString("cassandra-consistency")

		hosts := strings.Split(cassandraHosts, ",")
		for i := range hosts {
			hosts[i] = strings.TrimSpace(hosts[i])
		}

		cfg := cassandra.Config{
			Hosts:       hosts,
			Keyspace:    cassandraKeyspace,
			Username:    cassandraUsername,
			Password:    cassandraPassword,
			Consistency: cassandraConsistency,
			Migrate:     true,
		}
		store, err = cassandra.NewStore(context.Background(), cfg)

	case "memory":
		store = &memoryStoreWrapper{memory.NewStore()}

	default:
		return fmt.Errorf("unsupported storage type: %s", storageType)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to storage: %w", err)
	}
	defer store.Close()

	fmt.Println("Connected to storage successfully.")

	// Create auth service
	authService := auth.NewService(store)
	defer authService.Close()

	// Bootstrap admin user
	fmt.Printf("Bootstrapping admin user '%s'...\n", adminUsername)
	result, err := authService.BootstrapAdmin(ctx, adminUsername, adminPassword, adminEmail)
	if err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}

	if result.Created {
		fmt.Printf("\n✓ Admin user '%s' created successfully with role 'super_admin'\n", result.Username)
		fmt.Println("\nYou can now start the schema registry and authenticate with these credentials.")
	} else {
		fmt.Printf("\n⚠ %s\n", result.Message)
		fmt.Println("\nNo changes were made to the database.")
	}

	return nil
}

// memoryStoreWrapper wraps memory.Store to satisfy the interface.
type memoryStoreWrapper struct {
	*memory.Store
}

func (w *memoryStoreWrapper) Close() error {
	return w.Store.Close()
}
