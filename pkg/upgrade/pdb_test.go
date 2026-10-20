// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type testPDBLister struct {
	list            *policyv1.PodDisruptionBudgetList
	pods            *corev1.PodList
	err             error
	daemonSets      map[string]*appsv1.DaemonSet
	daemonSetErrors map[string]error
	daemonSetGets   map[string]int
}

func (l testPDBLister) List(context.Context, metav1.ListOptions) (*policyv1.PodDisruptionBudgetList, error) {
	return l.list, l.err
}

func (l testPDBLister) ListPods(context.Context, metav1.ListOptions) (*corev1.PodList, error) {
	if l.pods == nil {
		return &corev1.PodList{}, l.err
	}
	return l.pods, l.err
}

func (l testPDBLister) Get(_ context.Context, namespace, name string) (*appsv1.DaemonSet, error) {
	key := namespace + "/" + name
	if l.daemonSetGets != nil {
		l.daemonSetGets[key]++
	}
	if err, ok := l.daemonSetErrors[key]; ok {
		return nil, err
	}
	if daemonSet, ok := l.daemonSets[key]; ok {
		return daemonSet, nil
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{Group: "apps", Resource: "daemonsets"}, name)
}

func blockingTestPDB(allowed int32) policyv1.PodDisruptionBudget {
	return policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{Generation: 1, Namespace: "payments", Name: "api"},
		Spec: policyv1.PodDisruptionBudgetSpec{Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "api"},
		}},
		Status: policyv1.PodDisruptionBudgetStatus{
			ObservedGeneration: 1, ExpectedPods: 2, CurrentHealthy: 2, DesiredHealthy: 2,
			DisruptionsAllowed: allowed,
		},
	}
}

func candidateTestPod(name string) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "payments", Name: name, Labels: map[string]string{"app": "api"}},
		Spec:       corev1.PodSpec{NodeName: "node-a"},
		Status: corev1.PodStatus{Phase: corev1.PodRunning, Conditions: []corev1.PodCondition{{
			Type: corev1.PodReady, Status: corev1.ConditionTrue,
		}}},
	}
}

func daemonSetOwner(controller bool) metav1.OwnerReference {
	return metav1.OwnerReference{APIVersion: "apps/v1", Kind: "DaemonSet", Name: "agent", Controller: &controller}
}

func blockingLister(pods ...corev1.Pod) testPDBLister {
	return testPDBLister{
		list: &policyv1.PodDisruptionBudgetList{Items: []policyv1.PodDisruptionBudget{blockingTestPDB(0)}},
		pods: &corev1.PodList{Items: pods},
	}
}

var _ = Describe("Potential PDB blockers", func() {
	It("reports PDBs with no allowed disruptions in sorted order", func() {
		lister := testPDBLister{list: &policyv1.PodDisruptionBudgetList{Items: []policyv1.PodDisruptionBudget{
			{
				ObjectMeta: metav1.ObjectMeta{Generation: 1, Namespace: "zeta", Name: "allowed"},
				Status: policyv1.PodDisruptionBudgetStatus{
					ObservedGeneration: 1, ExpectedPods: 2, CurrentHealthy: 2, DesiredHealthy: 1, DisruptionsAllowed: 1,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Generation: 1, Namespace: "zeta", Name: "api"},
				Spec:       policyv1.PodDisruptionBudgetSpec{Selector: &metav1.LabelSelector{}},
				Status: policyv1.PodDisruptionBudgetStatus{
					ObservedGeneration: 1, ExpectedPods: 3, CurrentHealthy: 2, DesiredHealthy: 2, DisruptionsAllowed: 0,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Generation: 1, Namespace: "payments", Name: "api"},
				Spec:       policyv1.PodDisruptionBudgetSpec{Selector: &metav1.LabelSelector{}},
				Status: policyv1.PodDisruptionBudgetStatus{
					ObservedGeneration: 1, ExpectedPods: 3, CurrentHealthy: 2, DesiredHealthy: 2, DisruptionsAllowed: 0,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Generation: 1, Namespace: "empty", Name: "unused"},
				Status:     policyv1.PodDisruptionBudgetStatus{ObservedGeneration: 1},
			},
		}}, pods: &corev1.PodList{Items: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{Namespace: "zeta", Name: "zeta-api"},
				Spec:       corev1.PodSpec{NodeName: "node-a"},
				Status: corev1.PodStatus{Phase: corev1.PodRunning, Conditions: []corev1.PodCondition{{
					Type: corev1.PodReady, Status: corev1.ConditionTrue,
				}}},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Namespace: "payments", Name: "payments-api"},
				Spec:       corev1.PodSpec{NodeName: "node-b"},
				Status: corev1.PodStatus{Phase: corev1.PodRunning, Conditions: []corev1.PodCondition{{
					Type: corev1.PodReady, Status: corev1.ConditionTrue,
				}}},
			},
		}}}

		got, err := checkPotentialPDBBlockers(context.Background(), lister, lister, lister)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal([]PDBBlocker{
			{
				Namespace: "payments", Name: "api", ExpectedPods: 3, CurrentHealthy: 2, DesiredHealthy: 2,
				Budget: "minAvailable/maxUnavailable not set",
				Reasons: []string{
					"node 'node-b' has 1 selected pod(s), but only 0 disruption(s) are currently allowed",
					"the PDB currently allows zero disruptions",
				},
				MatchingPods: []string{"payments/payments-api"}, Nodes: []string{"node-b"},
			},
			{
				Namespace: "zeta", Name: "api", ExpectedPods: 3, CurrentHealthy: 2, DesiredHealthy: 2,
				Budget: "minAvailable/maxUnavailable not set",
				Reasons: []string{
					"node 'node-a' has 1 selected pod(s), but only 0 disruption(s) are currently allowed",
					"the PDB currently allows zero disruptions",
				},
				MatchingPods: []string{"zeta/zeta-api"}, Nodes: []string{"node-a"},
			},
		}))
	})

	It("rejects a kubeconfig API server that differs from the selected cluster", func() {
		Expect(sameAPIServer("https://api.example.com:6443", "https://other.example.com:6443")).To(BeFalse())
	})

	It("reports blockers even when another PDB has stale status", func() {
		lister := testPDBLister{list: &policyv1.PodDisruptionBudgetList{Items: []policyv1.PodDisruptionBudget{
			{
				ObjectMeta: metav1.ObjectMeta{Generation: 2, Namespace: "stale", Name: "budget"},
				Status:     policyv1.PodDisruptionBudgetStatus{ObservedGeneration: 1, ExpectedPods: 3},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Generation: 1, Namespace: "payments", Name: "api"},
				Spec:       policyv1.PodDisruptionBudgetSpec{Selector: &metav1.LabelSelector{}},
				Status: policyv1.PodDisruptionBudgetStatus{
					ObservedGeneration: 1, ExpectedPods: 3, CurrentHealthy: 2, DesiredHealthy: 2, DisruptionsAllowed: 0,
				},
			},
		}}, pods: &corev1.PodList{Items: []corev1.Pod{{
			ObjectMeta: metav1.ObjectMeta{Namespace: "payments", Name: "payments-api"},
			Spec:       corev1.PodSpec{NodeName: "node-b"},
			Status: corev1.PodStatus{Phase: corev1.PodRunning, Conditions: []corev1.PodCondition{{
				Type: corev1.PodReady, Status: corev1.ConditionTrue,
			}}},
		}}}}

		got, err := checkPotentialPDBBlockers(context.Background(), lister, lister, lister)
		Expect(err).To(MatchError(ContainSubstring("has not observed generation 2")))
		Expect(got).To(Equal([]PDBBlocker{{
			Namespace: "payments", Name: "api", ExpectedPods: 3, CurrentHealthy: 2, DesiredHealthy: 2,
			Budget: "minAvailable/maxUnavailable not set",
			Reasons: []string{
				"node 'node-b' has 1 selected pod(s), but only 0 disruption(s) are currently allowed",
				"the PDB currently allows zero disruptions",
			},
			MatchingPods: []string{"payments/payments-api"}, Nodes: []string{"node-b"},
		}}))
	})

	It("skips pods controlled by an existing DaemonSet", func() {
		pod := candidateTestPod("agent-abc")
		pod.OwnerReferences = []metav1.OwnerReference{daemonSetOwner(true)}
		lister := blockingLister(pod)
		lister.daemonSets = map[string]*appsv1.DaemonSet{"payments/agent": {}}
		lister.daemonSetGets = make(map[string]int)

		got, err := checkPotentialPDBBlockers(context.Background(), lister, lister, lister)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(BeEmpty())
		Expect(lister.daemonSetGets["payments/agent"]).To(Equal(1))
	})

	It("includes orphaned DaemonSet pods as drain candidates", func() {
		pod := candidateTestPod("agent-abc")
		pod.OwnerReferences = []metav1.OwnerReference{daemonSetOwner(true)}
		lister := blockingLister(pod)
		lister.daemonSetErrors = map[string]error{
			"payments/agent": apierrors.NewNotFound(schema.GroupResource{Group: "apps", Resource: "daemonsets"}, "agent"),
		}
		lister.daemonSetGets = make(map[string]int)

		got, err := checkPotentialPDBBlockers(context.Background(), lister, lister, lister)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(HaveLen(1))
		Expect(got[0].MatchingPods).To(Equal([]string{"payments/agent-abc"}))
		Expect(lister.daemonSetGets["payments/agent"]).To(Equal(1))
	})

	It("treats non-controller DaemonSet owner references as drain candidates", func() {
		pod := candidateTestPod("orphan-like")
		pod.OwnerReferences = []metav1.OwnerReference{daemonSetOwner(false)}
		lister := blockingLister(pod)
		lister.daemonSetGets = make(map[string]int)

		got, err := checkPotentialPDBBlockers(context.Background(), lister, lister, lister)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(HaveLen(1))
		Expect(lister.daemonSetGets).To(BeEmpty())
	})

	It("returns an inconclusive error for denied DaemonSet access while preserving other findings", func() {
		ordinaryPod := candidateTestPod("api-0")
		daemonSetPod := candidateTestPod("agent-abc")
		daemonSetPod.OwnerReferences = []metav1.OwnerReference{daemonSetOwner(true)}
		lister := blockingLister(ordinaryPod, daemonSetPod)
		lister.daemonSetErrors = map[string]error{
			"payments/agent": apierrors.NewForbidden(schema.GroupResource{Group: "apps", Resource: "daemonsets"}, "agent", fmt.Errorf("forbidden")),
		}
		lister.daemonSetGets = make(map[string]int)

		got, err := checkPotentialPDBBlockers(context.Background(), lister, lister, lister)
		Expect(err).To(MatchError(ContainSubstring("requires get access to daemonsets.apps")))
		Expect(got).To(HaveLen(1))
		Expect(got[0].MatchingPods).To(Equal([]string{"payments/api-0"}))
		Expect(lister.daemonSetGets["payments/agent"]).To(Equal(1))
	})

	It("caches DaemonSet lookups for multiple orphaned pods", func() {
		first := candidateTestPod("agent-a")
		first.OwnerReferences = []metav1.OwnerReference{daemonSetOwner(true)}
		second := candidateTestPod("agent-b")
		second.OwnerReferences = []metav1.OwnerReference{daemonSetOwner(true)}
		lister := blockingLister(first, second)
		lister.daemonSetErrors = map[string]error{
			"payments/agent": apierrors.NewNotFound(schema.GroupResource{Group: "apps", Resource: "daemonsets"}, "agent"),
		}
		lister.daemonSetGets = make(map[string]int)

		got, err := checkPotentialPDBBlockers(context.Background(), lister, lister, lister)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(HaveLen(1))
		Expect(got[0].MatchingPods).To(Equal([]string{"payments/agent-a", "payments/agent-b"}))
		Expect(lister.daemonSetGets["payments/agent"]).To(Equal(1))
	})

	It("counts a pod still listed in DisruptedPods but excludes deleting pods", func() {
		active := candidateTestPod("disrupted")
		deleting := candidateTestPod("deleting")
		now := metav1.Now()
		deleting.DeletionTimestamp = &now
		lister := blockingLister(active, deleting)
		pdb := blockingTestPDB(0)
		pdb.Status.DisruptedPods = map[string]metav1.Time{"disrupted": metav1.Now()}
		lister.list.Items[0] = pdb

		got, err := checkPotentialPDBBlockers(context.Background(), lister, lister, lister)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(HaveLen(1))
		Expect(got[0].MatchingPods).To(Equal([]string{"payments/disrupted"}))
	})

	It("reports the API server DisruptedPods size limit even when disruptions are allowed", func() {
		lister := blockingLister(candidateTestPod("api-0"))
		pdb := blockingTestPDB(1)
		pdb.Status.DisruptedPods = make(map[string]metav1.Time, maxDisruptedPods+1)
		for i := 0; i <= maxDisruptedPods; i++ {
			pdb.Status.DisruptedPods[fmt.Sprintf("pod-%d", i)] = metav1.Now()
		}
		lister.list.Items[0] = pdb

		got, err := checkPotentialPDBBlockers(context.Background(), lister, lister, lister)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(HaveLen(1))
		Expect(got[0].Reasons).To(ContainElement(ContainSubstring("exceeding the API server limit")))
	})

})
