package accessrequests

import (
	"context"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/openshift-online/ocm-sdk-go/accesstransparency/v1"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/output"
	. "github.com/openshift/rosa/pkg/test"
)

var (
	accessRequestsOutput = `STATE     ID         CLUSTER ID         UPDATED AT
Approved  mock-id-1  mock-cluster-id-1  Sat Sep 28 20:35:00 UTC 2024
Pending   mock-id-2  mock-cluster-id-2  Sat Sep 28 20:35:00 UTC 2024
INFO: Run the following command to approve or deny the Access Request:

   rosa create decision --access-request <ID> --decision Approved
   rosa create decision --access-request <ID> --decision Denied --justification "justification"

`
	clusterAccessRequestsOutput = `STATE    ID         CLUSTER ID       UPDATED AT
Pending  mock-id-1  mock-cluster-id  Sat Sep 28 21:35:00 UTC 2024
Denied   mock-id-2  mock-cluster-id  Sat Sep 28 20:35:00 UTC 2024
INFO: Run the following command to approve or deny the Access Request:

   rosa create decision --access-request mock-id-1 --decision Approved
   rosa create decision --access-request mock-id-1 --decision Denied --justification "justification"

`
)

var _ = Describe("rosa attach policy", func() {
	Context("Create Command", func() {
		It("Returns Command", func() {

			cmd := NewListAccessRequestsCommand()
			Expect(cmd).NotTo(BeNil())

			Expect(cmd.Use).To(Equal(use))
			Expect(cmd.Example).To(Equal(example))
			Expect(cmd.Short).To(Equal(short))
			Expect(cmd.Long).To(Equal(long))
			Expect(cmd.Args).NotTo(BeNil())
			Expect(cmd.Run).NotTo(BeNil())

			flag := cmd.Flags().Lookup("cluster")
			Expect(flag).NotTo(BeNil())
		})
	})

	Context("Command Runner", func() {

		var (
			t *TestingRuntime
			c *cobra.Command
		)

		BeforeEach(func() {
			t = NewTestRuntime()
			c = NewListAccessRequestsCommand()
			output.SetOutput("")
		})

		AfterEach(func() {
			output.SetOutput("")
		})

		It("Returns an error if OCM API fails to list Access Requests", func() {
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusInternalServerError, "{}"))

			runner := ListAccessRequestsRunner()
			err := runner(nil, t.RosaRuntime, c, nil)

			Expect(err).NotTo(BeNil())
		})

		It("Prints message if there are no Access Requests in Approved/Pending", func() {
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatAccessRequestList([]*v1.AccessRequest{})))

			runner := ListAccessRequestsRunner()

			t.StdOutReader.Record()
			err := runner(context.Background(), t.RosaRuntime, c, nil)
			Expect(err).NotTo(HaveOccurred())

			stdOut, _ := t.StdOutReader.Read()
			Expect(stdOut).To(Equal("INFO: There are no Access Requests in Pending or Approved status.\n"))
		})

		It("Prints list of all Access Requests", func() {
			layout := "Jan 2, 2006 at 3:04pm (MST)"
			mock_time, _ := time.Parse(layout, "Sep 28, 2024 at 8:35pm (PST)")
			accessRequest1, err := v1.NewAccessRequest().
				ID("mock-id-1").
				ClusterId("mock-cluster-id-1").
				UpdatedAt(mock_time).
				Status(v1.NewAccessRequestStatus().State(v1.AccessRequestStateApproved)).
				Build()
			Expect(err).NotTo(HaveOccurred())
			accessRequest2, err := v1.NewAccessRequest().
				ID("mock-id-2").
				ClusterId("mock-cluster-id-2").
				UpdatedAt(mock_time).
				Status(v1.NewAccessRequestStatus().State(v1.AccessRequestStatePending)).
				Build()
			Expect(err).NotTo(HaveOccurred())

			t.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, FormatAccessRequestList(
						[]*v1.AccessRequest{accessRequest1, accessRequest2})))

			runner := ListAccessRequestsRunner()

			t.StdOutReader.Record()
			err = runner(context.Background(), t.RosaRuntime, c, nil)
			Expect(err).NotTo(HaveOccurred())

			stdOut, _ := t.StdOutReader.Read()
			fmt.Println(stdOut)
			Expect(stdOut).To(Equal(accessRequestsOutput))
		})

		It("Prints list of all Access Requests for specified cluster", func() {
			layout := "Jan 2, 2006 at 3:04pm (MST)"
			mock_time_1, _ := time.Parse(layout, "Sep 28, 2024 at 9:35pm (PST)")
			mock_time_2, _ := time.Parse(layout, "Sep 28, 2024 at 8:35pm (PST)")
			accessRequest1, err := v1.NewAccessRequest().
				ID("mock-id-1").
				ClusterId("mock-cluster-id").
				UpdatedAt(mock_time_1).
				Status(v1.NewAccessRequestStatus().State(v1.AccessRequestStatePending)).
				Build()
			Expect(err).NotTo(HaveOccurred())
			accessRequest2, err := v1.NewAccessRequest().
				ID("mock-id-2").
				ClusterId("mock-cluster-id").
				UpdatedAt(mock_time_2).
				Status(v1.NewAccessRequestStatus().State(v1.AccessRequestStateDenied)).
				Build()
			Expect(err).NotTo(HaveOccurred())
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.ID("mock-cluster-id")
			})

			t.SetCluster(cluster.Name(), cluster)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, FormatAccessRequestList(
						[]*v1.AccessRequest{accessRequest1, accessRequest2})))

			runner := ListAccessRequestsRunner()

			t.StdOutReader.Record()
			c.Flags().Set("cluster", "mock-cluster-id")
			err = runner(context.Background(), t.RosaRuntime, c, nil)
			Expect(err).NotTo(HaveOccurred())

			stdOut, _ := t.StdOutReader.Read()
			Expect(stdOut).To(Equal(clusterAccessRequestsOutput))
		})

	})
})
