package operatorroles

import (
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
