package operatorroles

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/hyperfleet"
)

func TestWorkerTrustPolicyParses(t *testing.T) {
	doc := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":["ec2.amazonaws.com"]},"Action":"sts:AssumeRole"}]}`
	if _, err := aws.ParsePolicyDocument(doc); err != nil {
		t.Fatal(err)
	}
}

func TestGetHCPOperatorRolesMatchesSuffixes(t *testing.T) {
	roles := getHCPOperatorRoles()
	suffixes := hyperfleet.OperatorRoleSuffixes()
	if len(roles) != len(suffixes) {
		t.Fatalf("expected %d roles, got %d", len(suffixes), len(roles))
	}
	for i, suffix := range suffixes {
		if roles[i].Name != suffix {
			t.Fatalf("role[%d]: expected name %q, got %q", i, suffix, roles[i].Name)
		}
	}
}

func TestImageRegistryTrustPolicyUsesExactIssuerSubjectsAndAudience(t *testing.T) {
	const (
		issuerDomain = "oidc.example.com/us-east-1/issuer-id"
		accountID    = "123456789012"
	)

	var imageRegistryRole *operatorRoleSpec
	for _, role := range getHCPOperatorRoles() {
		if role.Name == hyperfleet.SuffixImageRegistry {
			imageRegistryRole = &role
			break
		}
	}
	if imageRegistryRole == nil {
		t.Fatal("image registry role was not configured")
	}

	policyJSON, err := buildOIDCTrustPolicy("aws", accountID, issuerDomain, imageRegistryRole.ServiceAccounts)
	if err != nil {
		t.Fatalf("building trust policy: %v", err)
	}

	var policy struct {
		Statement []struct {
			Principal struct {
				Federated string `json:"Federated"`
			} `json:"Principal"`
			Action    string `json:"Action"`
			Condition struct {
				StringEquals map[string]json.RawMessage `json:"StringEquals"`
			} `json:"Condition"`
		} `json:"Statement"`
	}
	if err := json.Unmarshal([]byte(policyJSON), &policy); err != nil {
		t.Fatalf("parsing generated trust policy: %v", err)
	}
	if len(policy.Statement) != 1 {
		t.Fatalf("expected one trust statement, got %d", len(policy.Statement))
	}
	statement := policy.Statement[0]
	if want := "arn:aws:iam::" + accountID + ":oidc-provider/" + issuerDomain; statement.Principal.Federated != want {
		t.Errorf("unexpected OIDC provider: got %q, want %q", statement.Principal.Federated, want)
	}
	if statement.Action != "sts:AssumeRoleWithWebIdentity" {
		t.Errorf("unexpected trust action: %q", statement.Action)
	}
	if got := len(statement.Condition.StringEquals); got != 2 {
		t.Errorf("expected only exact sub and aud conditions, got %d conditions", got)
	}

	var subjects []string
	if err := json.Unmarshal(statement.Condition.StringEquals[issuerDomain+":sub"], &subjects); err != nil {
		t.Fatalf("parsing service account subjects: %v", err)
	}
	wantSubjects := []string{
		"system:serviceaccount:openshift-image-registry:cluster-image-registry-operator",
		"system:serviceaccount:openshift-image-registry:registry",
	}
	if !reflect.DeepEqual(subjects, wantSubjects) {
		t.Errorf("unexpected service account subjects: got %v, want %v", subjects, wantSubjects)
	}

	var audience string
	if err := json.Unmarshal(statement.Condition.StringEquals[issuerDomain+":aud"], &audience); err != nil {
		t.Fatalf("parsing audience: %v", err)
	}
	if audience != "openshift" {
		t.Errorf("unexpected audience: got %q, want %q", audience, "openshift")
	}
}
