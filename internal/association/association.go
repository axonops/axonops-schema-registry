// Package association provides schema-to-resource associations.
package association

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AssociationType represents the type of association.
type AssociationType string

const (
	AssociationTypeStrong AssociationType = "strong" // Schema required for resource
	AssociationTypeWeak   AssociationType = "weak"   // Schema optional for resource
)

// ResourceType represents the type of resource being associated.
type ResourceType string

const (
	ResourceTypeTopic     ResourceType = "topic"
	ResourceTypeConnector ResourceType = "connector"
	ResourceTypeStream    ResourceType = "stream"
	ResourceTypeTable     ResourceType = "table"
	ResourceTypeCustom    ResourceType = "custom"
)

// Association represents a link between a schema and a resource.
type Association struct {
	ID              string            `json:"id"`
	Subject         string            `json:"subject"`
	Version         int               `json:"version"`
	ResourceType    ResourceType      `json:"resource_type"`
	ResourceName    string            `json:"resource_name"`
	AssociationType AssociationType   `json:"association_type"`
	Role            string            `json:"role,omitempty"` // key, value, header, etc.
	Metadata        map[string]string `json:"metadata,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// AssociationManager manages schema-resource associations.
type AssociationManager struct {
	mu           sync.RWMutex
	associations map[string]*Association
	bySubject    map[string][]string // subject -> association IDs
	byResource   map[string][]string // resourceType:resourceName -> association IDs
}

// NewAssociationManager creates a new association manager.
func NewAssociationManager() *AssociationManager {
	return &AssociationManager{
		associations: make(map[string]*Association),
		bySubject:    make(map[string][]string),
		byResource:   make(map[string][]string),
	}
}

// CreateAssociation creates a new association.
func (am *AssociationManager) CreateAssociation(assoc *Association) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if assoc.Subject == "" {
		return fmt.Errorf("subject is required")
	}
	if assoc.ResourceName == "" {
		return fmt.Errorf("resource name is required")
	}
	if assoc.ResourceType == "" {
		assoc.ResourceType = ResourceTypeCustom
	}
	if assoc.AssociationType == "" {
		assoc.AssociationType = AssociationTypeWeak
	}

	// Generate ID if not provided
	if assoc.ID == "" {
		assoc.ID = uuid.New().String()
	}

	now := time.Now()
	assoc.CreatedAt = now
	assoc.UpdatedAt = now

	// Check for duplicate
	resourceKey := am.resourceKey(assoc.ResourceType, assoc.ResourceName)
	for _, id := range am.bySubject[assoc.Subject] {
		existing := am.associations[id]
		if existing.ResourceType == assoc.ResourceType &&
			existing.ResourceName == assoc.ResourceName &&
			existing.Role == assoc.Role {
			return fmt.Errorf("association already exists")
		}
	}

	am.associations[assoc.ID] = assoc
	am.bySubject[assoc.Subject] = append(am.bySubject[assoc.Subject], assoc.ID)
	am.byResource[resourceKey] = append(am.byResource[resourceKey], assoc.ID)

	return nil
}

// GetAssociation retrieves an association by ID.
func (am *AssociationManager) GetAssociation(id string) (*Association, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	assoc, exists := am.associations[id]
	if !exists {
		return nil, fmt.Errorf("association not found: %s", id)
	}
	return assoc, nil
}

// UpdateAssociation updates an existing association.
func (am *AssociationManager) UpdateAssociation(assoc *Association) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	existing, exists := am.associations[assoc.ID]
	if !exists {
		return fmt.Errorf("association not found: %s", assoc.ID)
	}

	// Update fields
	assoc.CreatedAt = existing.CreatedAt
	assoc.UpdatedAt = time.Now()

	// If subject or resource changed, update indexes
	if existing.Subject != assoc.Subject {
		am.removeFromSubjectIndex(existing.Subject, assoc.ID)
		am.bySubject[assoc.Subject] = append(am.bySubject[assoc.Subject], assoc.ID)
	}

	oldResourceKey := am.resourceKey(existing.ResourceType, existing.ResourceName)
	newResourceKey := am.resourceKey(assoc.ResourceType, assoc.ResourceName)
	if oldResourceKey != newResourceKey {
		am.removeFromResourceIndex(oldResourceKey, assoc.ID)
		am.byResource[newResourceKey] = append(am.byResource[newResourceKey], assoc.ID)
	}

	am.associations[assoc.ID] = assoc
	return nil
}

// DeleteAssociation deletes an association.
func (am *AssociationManager) DeleteAssociation(id string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	assoc, exists := am.associations[id]
	if !exists {
		return fmt.Errorf("association not found: %s", id)
	}

	am.removeFromSubjectIndex(assoc.Subject, id)
	am.removeFromResourceIndex(am.resourceKey(assoc.ResourceType, assoc.ResourceName), id)
	delete(am.associations, id)

	return nil
}

// ListAssociations returns all associations.
func (am *AssociationManager) ListAssociations() []*Association {
	am.mu.RLock()
	defer am.mu.RUnlock()

	result := make([]*Association, 0, len(am.associations))
	for _, assoc := range am.associations {
		result = append(result, assoc)
	}
	return result
}

// GetBySubject returns associations for a subject.
func (am *AssociationManager) GetBySubject(subject string) []*Association {
	am.mu.RLock()
	defer am.mu.RUnlock()

	ids := am.bySubject[subject]
	result := make([]*Association, 0, len(ids))
	for _, id := range ids {
		if assoc, ok := am.associations[id]; ok {
			result = append(result, assoc)
		}
	}
	return result
}

// GetByResource returns associations for a resource.
func (am *AssociationManager) GetByResource(resourceType ResourceType, resourceName string) []*Association {
	am.mu.RLock()
	defer am.mu.RUnlock()

	key := am.resourceKey(resourceType, resourceName)
	ids := am.byResource[key]
	result := make([]*Association, 0, len(ids))
	for _, id := range ids {
		if assoc, ok := am.associations[id]; ok {
			result = append(result, assoc)
		}
	}
	return result
}

// GetStrongAssociations returns only strong associations for a resource.
func (am *AssociationManager) GetStrongAssociations(resourceType ResourceType, resourceName string) []*Association {
	all := am.GetByResource(resourceType, resourceName)
	result := make([]*Association, 0)
	for _, assoc := range all {
		if assoc.AssociationType == AssociationTypeStrong {
			result = append(result, assoc)
		}
	}
	return result
}

// BatchCreate creates multiple associations.
func (am *AssociationManager) BatchCreate(associations []*Association) error {
	for _, assoc := range associations {
		if err := am.CreateAssociation(assoc); err != nil {
			return fmt.Errorf("failed to create association %s: %w", assoc.ID, err)
		}
	}
	return nil
}

// BatchDelete deletes multiple associations.
func (am *AssociationManager) BatchDelete(ids []string) error {
	for _, id := range ids {
		if err := am.DeleteAssociation(id); err != nil {
			return fmt.Errorf("failed to delete association %s: %w", id, err)
		}
	}
	return nil
}

// resourceKey generates a unique key for a resource.
func (am *AssociationManager) resourceKey(resourceType ResourceType, resourceName string) string {
	return string(resourceType) + ":" + resourceName
}

// removeFromSubjectIndex removes an association ID from the subject index.
func (am *AssociationManager) removeFromSubjectIndex(subject, id string) {
	ids := am.bySubject[subject]
	for i, existingID := range ids {
		if existingID == id {
			am.bySubject[subject] = append(ids[:i], ids[i+1:]...)
			break
		}
	}
}

// removeFromResourceIndex removes an association ID from the resource index.
func (am *AssociationManager) removeFromResourceIndex(key, id string) {
	ids := am.byResource[key]
	for i, existingID := range ids {
		if existingID == id {
			am.byResource[key] = append(ids[:i], ids[i+1:]...)
			break
		}
	}
}

// LifecyclePolicy defines retention and migration rules for associations.
type LifecyclePolicy struct {
	Name              string        `json:"name"`
	RetentionDays     int           `json:"retention_days,omitempty"`
	DeleteOnOrphan    bool          `json:"delete_on_orphan"` // Delete when schema is deleted
	MigrateOnUpdate   bool          `json:"migrate_on_update"` // Update version when schema updated
	VersionConstraint string        `json:"version_constraint,omitempty"` // e.g., "latest", ">=1"
}

// PolicyManager manages lifecycle policies.
type PolicyManager struct {
	mu       sync.RWMutex
	policies map[string]*LifecyclePolicy
}

// NewPolicyManager creates a new policy manager.
func NewPolicyManager() *PolicyManager {
	return &PolicyManager{
		policies: make(map[string]*LifecyclePolicy),
	}
}

// SetPolicy sets a lifecycle policy.
func (pm *PolicyManager) SetPolicy(policy *LifecyclePolicy) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.policies[policy.Name] = policy
}

// GetPolicy retrieves a lifecycle policy.
func (pm *PolicyManager) GetPolicy(name string) (*LifecyclePolicy, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	policy, ok := pm.policies[name]
	return policy, ok
}

// ListPolicies returns all policies.
func (pm *PolicyManager) ListPolicies() []*LifecyclePolicy {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	result := make([]*LifecyclePolicy, 0, len(pm.policies))
	for _, policy := range pm.policies {
		result = append(result, policy)
	}
	return result
}
