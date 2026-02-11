package association

import (
	"testing"
)

func TestNewAssociationManager(t *testing.T) {
	am := NewAssociationManager()
	if am == nil {
		t.Fatal("expected non-nil manager")
	}
	if len(am.ListAssociations()) != 0 {
		t.Error("expected 0 associations")
	}
}

func TestCreateAssociation(t *testing.T) {
	am := NewAssociationManager()

	assoc := &Association{
		Subject:      "test-subject",
		Version:      1,
		ResourceType: ResourceTypeTopic,
		ResourceName: "my-topic",
	}
	err := am.CreateAssociation(assoc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if assoc.ID == "" {
		t.Error("expected ID to be generated")
	}
	if assoc.AssociationType != AssociationTypeWeak {
		t.Errorf("expected default weak type, got %s", assoc.AssociationType)
	}
	if assoc.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestCreateAssociation_EmptySubject(t *testing.T) {
	am := NewAssociationManager()
	err := am.CreateAssociation(&Association{ResourceName: "r"})
	if err == nil {
		t.Error("expected error for empty subject")
	}
}

func TestCreateAssociation_EmptyResourceName(t *testing.T) {
	am := NewAssociationManager()
	err := am.CreateAssociation(&Association{Subject: "s"})
	if err == nil {
		t.Error("expected error for empty resource name")
	}
}

func TestCreateAssociation_DefaultResourceType(t *testing.T) {
	am := NewAssociationManager()
	assoc := &Association{Subject: "s", ResourceName: "r"}
	am.CreateAssociation(assoc)
	if assoc.ResourceType != ResourceTypeCustom {
		t.Errorf("expected default custom type, got %s", assoc.ResourceType)
	}
}

func TestCreateAssociation_Duplicate(t *testing.T) {
	am := NewAssociationManager()

	a1 := &Association{
		Subject:      "s",
		ResourceType: ResourceTypeTopic,
		ResourceName: "t",
		Role:         "key",
	}
	am.CreateAssociation(a1)

	a2 := &Association{
		Subject:      "s",
		ResourceType: ResourceTypeTopic,
		ResourceName: "t",
		Role:         "key",
	}
	err := am.CreateAssociation(a2)
	if err == nil {
		t.Error("expected duplicate error")
	}
}

func TestCreateAssociation_SameSubjectDifferentRole(t *testing.T) {
	am := NewAssociationManager()

	am.CreateAssociation(&Association{Subject: "s", ResourceType: ResourceTypeTopic, ResourceName: "t", Role: "key"})
	err := am.CreateAssociation(&Association{Subject: "s", ResourceType: ResourceTypeTopic, ResourceName: "t", Role: "value"})
	if err != nil {
		t.Fatalf("different roles should not conflict: %v", err)
	}
}

func TestCreateAssociation_PreservesExplicitID(t *testing.T) {
	am := NewAssociationManager()
	assoc := &Association{ID: "custom-id", Subject: "s", ResourceName: "r"}
	am.CreateAssociation(assoc)
	if assoc.ID != "custom-id" {
		t.Errorf("expected custom-id, got %s", assoc.ID)
	}
}

func TestGetAssociation(t *testing.T) {
	am := NewAssociationManager()
	assoc := &Association{Subject: "s", ResourceName: "r"}
	am.CreateAssociation(assoc)

	found, err := am.GetAssociation(assoc.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.Subject != "s" {
		t.Errorf("expected subject 's', got %s", found.Subject)
	}
}

func TestGetAssociation_NotFound(t *testing.T) {
	am := NewAssociationManager()
	_, err := am.GetAssociation("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent association")
	}
}

func TestUpdateAssociation(t *testing.T) {
	am := NewAssociationManager()
	assoc := &Association{Subject: "s", ResourceName: "r", ResourceType: ResourceTypeTopic}
	am.CreateAssociation(assoc)
	createdAt := assoc.CreatedAt

	updated := &Association{
		ID:           assoc.ID,
		Subject:      "s2",
		ResourceName: "r2",
		ResourceType: ResourceTypeConnector,
	}
	err := am.UpdateAssociation(updated)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.CreatedAt != createdAt {
		t.Error("CreatedAt should be preserved")
	}

	found, _ := am.GetAssociation(assoc.ID)
	if found.Subject != "s2" {
		t.Errorf("expected updated subject, got %s", found.Subject)
	}
}

func TestUpdateAssociation_NotFound(t *testing.T) {
	am := NewAssociationManager()
	err := am.UpdateAssociation(&Association{ID: "nonexistent"})
	if err == nil {
		t.Error("expected error")
	}
}

func TestUpdateAssociation_SubjectChange_UpdatesIndex(t *testing.T) {
	am := NewAssociationManager()
	assoc := &Association{Subject: "old-subject", ResourceName: "r", ResourceType: ResourceTypeTopic}
	am.CreateAssociation(assoc)

	am.UpdateAssociation(&Association{
		ID:           assoc.ID,
		Subject:      "new-subject",
		ResourceName: "r",
		ResourceType: ResourceTypeTopic,
	})

	// Old subject should have no associations
	if len(am.GetBySubject("old-subject")) != 0 {
		t.Error("expected 0 associations for old subject")
	}
	// New subject should have the association
	if len(am.GetBySubject("new-subject")) != 1 {
		t.Error("expected 1 association for new subject")
	}
}

func TestUpdateAssociation_ResourceChange_UpdatesIndex(t *testing.T) {
	am := NewAssociationManager()
	assoc := &Association{Subject: "s", ResourceName: "old-resource", ResourceType: ResourceTypeTopic}
	am.CreateAssociation(assoc)

	am.UpdateAssociation(&Association{
		ID:           assoc.ID,
		Subject:      "s",
		ResourceName: "new-resource",
		ResourceType: ResourceTypeTopic,
	})

	if len(am.GetByResource(ResourceTypeTopic, "old-resource")) != 0 {
		t.Error("expected 0 associations for old resource")
	}
	if len(am.GetByResource(ResourceTypeTopic, "new-resource")) != 1 {
		t.Error("expected 1 association for new resource")
	}
}

func TestDeleteAssociation(t *testing.T) {
	am := NewAssociationManager()
	assoc := &Association{Subject: "s", ResourceName: "r", ResourceType: ResourceTypeTopic}
	am.CreateAssociation(assoc)

	err := am.DeleteAssociation(assoc.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = am.GetAssociation(assoc.ID)
	if err == nil {
		t.Error("expected not found after deletion")
	}

	// Verify indexes cleaned up
	if len(am.GetBySubject("s")) != 0 {
		t.Error("expected subject index cleaned up")
	}
	if len(am.GetByResource(ResourceTypeTopic, "r")) != 0 {
		t.Error("expected resource index cleaned up")
	}
}

func TestDeleteAssociation_NotFound(t *testing.T) {
	am := NewAssociationManager()
	err := am.DeleteAssociation("nonexistent")
	if err == nil {
		t.Error("expected error")
	}
}

func TestListAssociations(t *testing.T) {
	am := NewAssociationManager()
	am.CreateAssociation(&Association{Subject: "s1", ResourceName: "r1"})
	am.CreateAssociation(&Association{Subject: "s2", ResourceName: "r2"})
	am.CreateAssociation(&Association{Subject: "s3", ResourceName: "r3"})

	list := am.ListAssociations()
	if len(list) != 3 {
		t.Errorf("expected 3 associations, got %d", len(list))
	}
}

func TestGetBySubject(t *testing.T) {
	am := NewAssociationManager()
	am.CreateAssociation(&Association{Subject: "s1", ResourceName: "r1"})
	am.CreateAssociation(&Association{Subject: "s1", ResourceName: "r2"})
	am.CreateAssociation(&Association{Subject: "s2", ResourceName: "r3"})

	s1Assocs := am.GetBySubject("s1")
	if len(s1Assocs) != 2 {
		t.Errorf("expected 2 associations for s1, got %d", len(s1Assocs))
	}

	s2Assocs := am.GetBySubject("s2")
	if len(s2Assocs) != 1 {
		t.Errorf("expected 1 association for s2, got %d", len(s2Assocs))
	}
}

func TestGetBySubject_Empty(t *testing.T) {
	am := NewAssociationManager()
	assocs := am.GetBySubject("nonexistent")
	if len(assocs) != 0 {
		t.Errorf("expected 0 associations, got %d", len(assocs))
	}
}

func TestGetByResource(t *testing.T) {
	am := NewAssociationManager()
	am.CreateAssociation(&Association{Subject: "s1", ResourceName: "topic-1", ResourceType: ResourceTypeTopic})
	am.CreateAssociation(&Association{Subject: "s2", ResourceName: "topic-1", ResourceType: ResourceTypeTopic})
	am.CreateAssociation(&Association{Subject: "s3", ResourceName: "topic-2", ResourceType: ResourceTypeTopic})

	assocs := am.GetByResource(ResourceTypeTopic, "topic-1")
	if len(assocs) != 2 {
		t.Errorf("expected 2 associations, got %d", len(assocs))
	}
}

func TestGetStrongAssociations(t *testing.T) {
	am := NewAssociationManager()
	am.CreateAssociation(&Association{
		Subject: "s1", ResourceName: "t", ResourceType: ResourceTypeTopic,
		AssociationType: AssociationTypeStrong, Role: "key",
	})
	am.CreateAssociation(&Association{
		Subject: "s2", ResourceName: "t", ResourceType: ResourceTypeTopic,
		AssociationType: AssociationTypeWeak, Role: "value",
	})

	strong := am.GetStrongAssociations(ResourceTypeTopic, "t")
	if len(strong) != 1 {
		t.Errorf("expected 1 strong association, got %d", len(strong))
	}
	if strong[0].Subject != "s1" {
		t.Errorf("expected subject s1, got %s", strong[0].Subject)
	}
}

func TestBatchCreate(t *testing.T) {
	am := NewAssociationManager()
	assocs := []*Association{
		{Subject: "s1", ResourceName: "r1"},
		{Subject: "s2", ResourceName: "r2"},
		{Subject: "s3", ResourceName: "r3"},
	}
	err := am.BatchCreate(assocs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(am.ListAssociations()) != 3 {
		t.Errorf("expected 3 associations")
	}
}

func TestBatchCreate_StopsOnError(t *testing.T) {
	am := NewAssociationManager()
	assocs := []*Association{
		{Subject: "s1", ResourceName: "r1"},
		{Subject: "", ResourceName: "r2"}, // Invalid — no subject
		{Subject: "s3", ResourceName: "r3"},
	}
	err := am.BatchCreate(assocs)
	if err == nil {
		t.Error("expected error from batch create")
	}
	// First one should have been created
	if len(am.ListAssociations()) != 1 {
		t.Errorf("expected 1 association before error, got %d", len(am.ListAssociations()))
	}
}

func TestBatchDelete(t *testing.T) {
	am := NewAssociationManager()
	a1 := &Association{Subject: "s1", ResourceName: "r1"}
	a2 := &Association{Subject: "s2", ResourceName: "r2"}
	a3 := &Association{Subject: "s3", ResourceName: "r3"}
	am.CreateAssociation(a1)
	am.CreateAssociation(a2)
	am.CreateAssociation(a3)

	err := am.BatchDelete([]string{a1.ID, a3.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(am.ListAssociations()) != 1 {
		t.Errorf("expected 1 remaining, got %d", len(am.ListAssociations()))
	}
}

func TestBatchDelete_StopsOnError(t *testing.T) {
	am := NewAssociationManager()
	a1 := &Association{Subject: "s1", ResourceName: "r1"}
	am.CreateAssociation(a1)

	err := am.BatchDelete([]string{a1.ID, "nonexistent"})
	if err == nil {
		t.Error("expected error")
	}
}

// --- PolicyManager Tests ---

func TestNewPolicyManager(t *testing.T) {
	pm := NewPolicyManager()
	if pm == nil {
		t.Fatal("expected non-nil manager")
	}
	if len(pm.ListPolicies()) != 0 {
		t.Error("expected 0 policies")
	}
}

func TestSetPolicy(t *testing.T) {
	pm := NewPolicyManager()
	policy := &LifecyclePolicy{
		Name:           "default",
		RetentionDays:  30,
		DeleteOnOrphan: true,
	}
	pm.SetPolicy(policy)

	found, ok := pm.GetPolicy("default")
	if !ok {
		t.Fatal("expected to find policy")
	}
	if found.RetentionDays != 30 {
		t.Errorf("expected 30, got %d", found.RetentionDays)
	}
}

func TestGetPolicy_NotFound(t *testing.T) {
	pm := NewPolicyManager()
	_, ok := pm.GetPolicy("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestSetPolicy_Overwrite(t *testing.T) {
	pm := NewPolicyManager()
	pm.SetPolicy(&LifecyclePolicy{Name: "p", RetentionDays: 10})
	pm.SetPolicy(&LifecyclePolicy{Name: "p", RetentionDays: 20})

	found, _ := pm.GetPolicy("p")
	if found.RetentionDays != 20 {
		t.Errorf("expected 20, got %d", found.RetentionDays)
	}
}

func TestListPolicies(t *testing.T) {
	pm := NewPolicyManager()
	pm.SetPolicy(&LifecyclePolicy{Name: "a"})
	pm.SetPolicy(&LifecyclePolicy{Name: "b"})

	list := pm.ListPolicies()
	if len(list) != 2 {
		t.Errorf("expected 2 policies, got %d", len(list))
	}
}
