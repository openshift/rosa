/*
Copyright (c) 2026, Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package upgrade

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	coreclient "k8s.io/client-go/kubernetes/typed/core/v1"
	policyclient "k8s.io/client-go/kubernetes/typed/policy/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	pdbRequestTimeout = 30 * time.Second
	pdbListPageSize   = 500
	maxDisruptedPods  = 2000
)

// PDBBlocker describes a PDB that can prevent one or more current node drains.
type PDBBlocker struct {
	Namespace          string
	Name               string
	ExpectedPods       int32
	CurrentHealthy     int32
	DesiredHealthy     int32
	DisruptionsAllowed int32
	Budget             string
	Reasons            []string
	MatchingPods       []string
	Nodes              []string
}

type pdbLister interface {
	List(context.Context, metav1.ListOptions) (*policyv1.PodDisruptionBudgetList, error)
}

type podLister interface {
	ListPods(context.Context, metav1.ListOptions) (*corev1.PodList, error)
}

type daemonSetGetter interface {
	Get(context.Context, string, string) (*appsv1.DaemonSet, error)
}

type corePodLister struct {
	client coreclient.PodInterface
}

func (l corePodLister) ListPods(ctx context.Context, options metav1.ListOptions) (*corev1.PodList, error) {
	return l.client.List(ctx, options)
}

type appsDaemonSetGetter struct {
	client *http.Client
	apiURL string
}

func (g appsDaemonSetGetter) Get(ctx context.Context, namespace, name string) (*appsv1.DaemonSet, error) {
	server, err := url.Parse(g.apiURL)
	if err != nil {
		return nil, fmt.Errorf("could not parse Kubernetes API URL: %w", err)
	}
	server.Path = strings.TrimRight(server.Path, "/") + "/apis/apps/v1/namespaces/" + namespace + "/daemonsets/" + name
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("could not create DaemonSet request: %w", err)
	}
	response, err := g.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return nil, apierrors.NewNotFound(schema.GroupResource{Group: "apps", Resource: "daemonsets"}, name)
	}
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("Kubernetes API returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	daemonSet := &appsv1.DaemonSet{}
	if err := json.NewDecoder(response.Body).Decode(daemonSet); err != nil {
		return nil, fmt.Errorf("could not decode DaemonSet response: %w", err)
	}
	return daemonSet, nil
}

type daemonSetLookup struct {
	exists   bool
	err      error
	reported bool
}

type pdbAccumulator struct {
	blocker PDBBlocker
	reasons map[string]struct{}
	pods    map[string]struct{}
	nodes   map[string]struct{}
}

type observedPDB struct {
	pdb      policyv1.PodDisruptionBudget
	selector labels.Selector
}

// CheckPotentialPDBBlockers checks current PDB status and pod placement for node drains.
// It reads the user's current kubeconfig only when its API server matches the selected ROSA cluster.
func CheckPotentialPDBBlockers(ctx context.Context, clusterAPIURL string) ([]PDBBlocker, error) {
	if strings.TrimSpace(clusterAPIURL) == "" {
		return nil, fmt.Errorf("the selected ROSA cluster API URL is unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, pdbRequestTimeout)
	defer cancel()

	loadingConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
	config, err := loadingConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("could not load the current Kubernetes context for the PDB preflight: %w", err)
	}
	if !sameAPIServer(clusterAPIURL, config.Host) {
		return nil, fmt.Errorf("the current Kubernetes context is connected to %q, not the selected cluster API server %q",
			config.Host, clusterAPIURL)
	}
	config.Timeout = pdbRequestTimeout

	policyAPI, err := policyclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("could not create a Kubernetes API client for the PDB preflight: %w", err)
	}
	coreAPI, err := coreclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("could not create a Kubernetes API client for the pod preflight: %w", err)
	}
	transport, err := rest.TransportFor(config)
	if err != nil {
		return nil, fmt.Errorf("could not create a Kubernetes API transport for the DaemonSet preflight: %w", err)
	}

	return checkPotentialPDBBlockers(ctx,
		policyAPI.PodDisruptionBudgets(metav1.NamespaceAll),
		corePodLister{client: coreAPI.Pods(metav1.NamespaceAll)},
		appsDaemonSetGetter{client: &http.Client{Transport: transport, Timeout: pdbRequestTimeout}, apiURL: config.Host})
}

func checkPotentialPDBBlockers(ctx context.Context, pdbs pdbLister, pods podLister,
	daemonSets daemonSetGetter) ([]PDBBlocker, error) {
	pdbList, err := listAllPDBs(ctx, pdbs)
	if err != nil {
		return nil, fmt.Errorf("could not list Pod Disruption Budgets from the Kubernetes API (requires cluster-wide list access to poddisruptionbudgets.policy): %w", err)
	}

	accumulators := make(map[string]*pdbAccumulator)
	var scanErrors []error
	observed := make([]observedPDB, 0, len(pdbList))
	observedByKey := make(map[string]observedPDB, len(pdbList))
	observedByNamespace := make(map[string][]int)
	for _, pdb := range pdbList {
		if err := ctx.Err(); err != nil {
			return incompletePDBScan(accumulators, scanErrors, err)
		}
		if pdb.Status.ObservedGeneration != pdb.Generation || pdb.Generation == 0 {
			scanErrors = append(scanErrors, fmt.Errorf(
				"pod disruption budget '%s/%s' status has not observed generation %d",
				pdb.Namespace, pdb.Name, pdb.Generation))
			continue
		}

		// A nil selector in policy/v1 matches no pods; an empty selector matches all pods.
		selector := labels.Nothing()
		if pdb.Spec.Selector != nil {
			selector, err = metav1.LabelSelectorAsSelector(pdb.Spec.Selector)
			if err != nil {
				scanErrors = append(scanErrors, fmt.Errorf("pod disruption budget '%s/%s' has an invalid selector: %w",
					pdb.Namespace, pdb.Name, err))
				continue
			}
		}
		candidate := observedPDB{pdb: pdb, selector: selector}
		observed = append(observed, candidate)
		observedByKey[pdbKey(pdb)] = candidate
		observedByNamespace[pdb.Namespace] = append(observedByNamespace[pdb.Namespace], len(observed)-1)
	}

	podList, err := listAllPods(ctx, pods)
	if err != nil {
		scanErrors = append(scanErrors, fmt.Errorf("could not list pods from the Kubernetes API (requires cluster-wide list access to pods): %w", err))
		return finishBlockers(accumulators), errors.Join(scanErrors...)
	}

	podMatches := make(map[string][]int)
	pdbNodePods := make(map[string]map[string][]string)
	daemonSetCache := make(map[string]daemonSetLookup)
	for _, pod := range podList {
		if err := ctx.Err(); err != nil {
			return incompletePDBScan(accumulators, scanErrors, err)
		}
		if !isNodeDrainCandidate(pod) {
			continue
		}

		matchedIndexes := make([]int, 0)
		for _, index := range observedByNamespace[pod.Namespace] {
			if err := ctx.Err(); err != nil {
				return incompletePDBScan(accumulators, scanErrors, err)
			}
			candidate := observed[index]
			if candidate.pdb.Namespace != pod.Namespace || !candidate.selector.Matches(labels.Set(pod.Labels)) {
				continue
			}
			matchedIndexes = append(matchedIndexes, index)
		}
		if len(matchedIndexes) == 0 {
			continue
		}

		if controller := daemonSetController(&pod); controller != nil {
			key := pod.Namespace + "/" + controller.Name
			lookup, cached := daemonSetCache[key]
			if !cached {
				if daemonSets == nil {
					lookup.err = fmt.Errorf("no DaemonSet API client is available")
				} else {
					_, getErr := daemonSets.Get(ctx, pod.Namespace, controller.Name)
					switch {
					case getErr == nil:
						lookup.exists = true
					case apierrors.IsNotFound(getErr):
						lookup.exists = false
					default:
						lookup.err = getErr
					}
				}
				daemonSetCache[key] = lookup
			}
			if lookup.err != nil {
				if !lookup.reported {
					scanErrors = append(scanErrors, fmt.Errorf("could not verify whether pod '%s/%s' is managed by DaemonSet '%s/%s' (requires get access to daemonsets.apps in namespace '%s'): %w",
						pod.Namespace, pod.Name, pod.Namespace, controller.Name, pod.Namespace, lookup.err))
					lookup.reported = true
					daemonSetCache[key] = lookup
				}
				continue
			}
			if lookup.exists {
				continue
			}
		}

		podMatches[podKey(pod)] = matchedIndexes
		for _, index := range matchedIndexes {
			candidate := observed[index]
			if pdbAllowsUnhealthyPodEviction(candidate.pdb, pod) {
				continue
			}

			key := pdbKey(candidate.pdb)
			if pdbNodePods[key] == nil {
				pdbNodePods[key] = make(map[string][]string)
			}
			pdbNodePods[key][pod.Spec.NodeName] = append(pdbNodePods[key][pod.Spec.NodeName], pod.Namespace+"/"+pod.Name)
		}
	}

	for key, nodePods := range pdbNodePods {
		if err := ctx.Err(); err != nil {
			return incompletePDBScan(accumulators, scanErrors, err)
		}
		candidate := observedByKey[key]
		for node, selectedPods := range nodePods {
			if err := ctx.Err(); err != nil {
				return incompletePDBScan(accumulators, scanErrors, err)
			}
			count := len(selectedPods)
			disruptedPodsLimitExceeded := len(candidate.pdb.Status.DisruptedPods) > maxDisruptedPods
			if int32(count) <= candidate.pdb.Status.DisruptionsAllowed && !disruptedPodsLimitExceeded {
				continue
			}
			accumulator := getAccumulator(accumulators, candidate.pdb)
			accumulator.addNode(node)
			if candidate.pdb.Status.DisruptionsAllowed == 0 {
				accumulator.addReason("the PDB currently allows zero disruptions")
			}
			if int32(count) > candidate.pdb.Status.DisruptionsAllowed {
				accumulator.addReason(fmt.Sprintf("node '%s' has %d selected pod(s), but only %d disruption(s) are currently allowed",
					node, count, candidate.pdb.Status.DisruptionsAllowed))
			}
			if disruptedPodsLimitExceeded {
				accumulator.addReason(fmt.Sprintf("the PDB has %d pending disrupted pod entries, exceeding the API server limit of %d",
					len(candidate.pdb.Status.DisruptedPods), maxDisruptedPods))
			}
			for _, pod := range selectedPods {
				accumulator.addPod(pod)
			}
		}
	}

	for _, pod := range podList {
		if err := ctx.Err(); err != nil {
			return incompletePDBScan(accumulators, scanErrors, err)
		}
		indexes := podMatches[podKey(pod)]
		if len(indexes) < 2 {
			continue
		}
		matchedBudgets := make([]string, 0, len(indexes))
		for _, index := range indexes {
			matchedBudgets = append(matchedBudgets, pdbKey(observed[index].pdb))
		}
		sort.Strings(matchedBudgets)
		reason := fmt.Sprintf("pod '%s/%s' on node '%s' matches multiple PDBs (%s)",
			pod.Namespace, pod.Name, pod.Spec.NodeName, strings.Join(matchedBudgets, ", "))
		for _, index := range indexes {
			accumulator := getAccumulator(accumulators, observed[index].pdb)
			accumulator.addNode(pod.Spec.NodeName)
			accumulator.addPod(pod.Namespace + "/" + pod.Name)
			accumulator.addReason(reason)
		}
	}

	if err := ctx.Err(); err != nil {
		return incompletePDBScan(accumulators, scanErrors, err)
	}
	return finishBlockers(accumulators), errors.Join(scanErrors...)
}

func incompletePDBScan(accumulators map[string]*pdbAccumulator, scanErrors []error, cause error) ([]PDBBlocker, error) {
	scanErrors = append(scanErrors, fmt.Errorf("PDB preflight scan did not complete: %w", cause))
	return finishBlockers(accumulators), errors.Join(scanErrors...)
}

func listAllPDBs(ctx context.Context, lister pdbLister) ([]policyv1.PodDisruptionBudget, error) {
	options := metav1.ListOptions{Limit: pdbListPageSize}
	items := make([]policyv1.PodDisruptionBudget, 0)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		page, err := lister.List(ctx, options)
		if err != nil {
			return nil, err
		}
		items = append(items, page.Items...)
		if page.Continue == "" {
			return items, nil
		}
		options.Continue = page.Continue
	}
}

func listAllPods(ctx context.Context, lister podLister) ([]corev1.Pod, error) {
	options := metav1.ListOptions{Limit: pdbListPageSize}
	items := make([]corev1.Pod, 0)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		page, err := lister.ListPods(ctx, options)
		if err != nil {
			return nil, err
		}
		items = append(items, page.Items...)
		if page.Continue == "" {
			return items, nil
		}
		options.Continue = page.Continue
	}
}

func isNodeDrainCandidate(pod corev1.Pod) bool {
	if pod.Spec.NodeName == "" || pod.DeletionTimestamp != nil ||
		pod.Status.Phase == corev1.PodPending || pod.Status.Phase == corev1.PodSucceeded ||
		pod.Status.Phase == corev1.PodFailed {
		return false
	}
	if _, mirrorPod := pod.Annotations[corev1.MirrorPodAnnotationKey]; mirrorPod {
		return false
	}
	return true
}

func daemonSetController(pod *corev1.Pod) *metav1.OwnerReference {
	controller := metav1.GetControllerOf(pod)
	if controller == nil || controller.APIVersion != "apps/v1" || controller.Kind != "DaemonSet" {
		return nil
	}
	return controller
}

func pdbAllowsUnhealthyPodEviction(pdb policyv1.PodDisruptionBudget, pod corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning || podIsHealthy(pod) {
		return false
	}
	if pdb.Spec.UnhealthyPodEvictionPolicy != nil {
		switch *pdb.Spec.UnhealthyPodEvictionPolicy {
		case policyv1.AlwaysAllow:
			return true
		case policyv1.IfHealthyBudget:
			return pdb.Status.DesiredHealthy > 0 && pdb.Status.CurrentHealthy >= pdb.Status.DesiredHealthy
		default:
			return false
		}
	}
	return pdb.Status.DesiredHealthy > 0 && pdb.Status.CurrentHealthy >= pdb.Status.DesiredHealthy
}

func podIsHealthy(pod corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func getAccumulator(accumulators map[string]*pdbAccumulator, pdb policyv1.PodDisruptionBudget) *pdbAccumulator {
	key := pdbKey(pdb)
	if accumulator := accumulators[key]; accumulator != nil {
		return accumulator
	}
	accumulator := &pdbAccumulator{
		blocker: PDBBlocker{
			Namespace:          pdb.Namespace,
			Name:               pdb.Name,
			ExpectedPods:       pdb.Status.ExpectedPods,
			CurrentHealthy:     pdb.Status.CurrentHealthy,
			DesiredHealthy:     pdb.Status.DesiredHealthy,
			DisruptionsAllowed: pdb.Status.DisruptionsAllowed,
			Budget:             describeBudget(pdb),
		},
		reasons: make(map[string]struct{}),
		pods:    make(map[string]struct{}),
		nodes:   make(map[string]struct{}),
	}
	accumulators[key] = accumulator
	return accumulator
}

func (a *pdbAccumulator) addReason(reason string) {
	a.reasons[reason] = struct{}{}
}

func (a *pdbAccumulator) addPod(pod string) {
	a.pods[pod] = struct{}{}
}

func (a *pdbAccumulator) addNode(node string) {
	a.nodes[node] = struct{}{}
}

func finishBlockers(accumulators map[string]*pdbAccumulator) []PDBBlocker {
	blockers := make([]PDBBlocker, 0, len(accumulators))
	for _, accumulator := range accumulators {
		accumulator.blocker.Reasons = sortedKeys(accumulator.reasons)
		accumulator.blocker.MatchingPods = sortedKeys(accumulator.pods)
		accumulator.blocker.Nodes = sortedKeys(accumulator.nodes)
		blockers = append(blockers, accumulator.blocker)
	}
	sort.Slice(blockers, func(i, j int) bool {
		if blockers[i].Namespace == blockers[j].Namespace {
			return blockers[i].Name < blockers[j].Name
		}
		return blockers[i].Namespace < blockers[j].Namespace
	})
	return blockers
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func pdbKey(pdb policyv1.PodDisruptionBudget) string {
	return pdb.Namespace + "/" + pdb.Name
}

func podKey(pod corev1.Pod) string {
	return pod.Namespace + "/" + pod.Name
}

func describeBudget(pdb policyv1.PodDisruptionBudget) string {
	if pdb.Spec.MinAvailable != nil {
		return "minAvailable=" + pdb.Spec.MinAvailable.String()
	}
	if pdb.Spec.MaxUnavailable != nil {
		return "maxUnavailable=" + pdb.Spec.MaxUnavailable.String()
	}
	return "minAvailable/maxUnavailable not set"
}

func sameAPIServer(expected, actual string) bool {
	expectedURL, err := parseAPIServerURL(expected)
	if err != nil {
		return false
	}
	actualURL, err := parseAPIServerURL(actual)
	if err != nil {
		return false
	}

	return strings.EqualFold(expectedURL.Scheme, actualURL.Scheme) &&
		strings.EqualFold(expectedURL.Hostname(), actualURL.Hostname()) &&
		apiServerPort(expectedURL) == apiServerPort(actualURL)
}

func parseAPIServerURL(value string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return nil, err
	}
	if parsed.Hostname() == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return nil, fmt.Errorf("invalid API server URL")
	}
	return parsed, nil
}

func apiServerPort(server *url.URL) string {
	if port := server.Port(); port != "" {
		return port
	}
	if strings.EqualFold(server.Scheme, "https") {
		return "443"
	}
	return "80"
}
