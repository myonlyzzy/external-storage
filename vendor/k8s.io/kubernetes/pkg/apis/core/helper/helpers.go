/*
Copyright 2014 The Kubernetes Authors.

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

package helper

import (
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/kubernetes/pkg/apis/core"
)

// IsHugePageResourceName returns true if the resource name has the huge page
// resource prefix.
func IsHugePageResourceName(name core.ResourceName) bool {
	return strings.HasPrefix(string(name), core.ResourceHugePagesPrefix)
}

// IsQuotaHugePageResourceName returns true if the resource name has the quota
// related huge page resource prefix.
func IsQuotaHugePageResourceName(name core.ResourceName) bool {
	return strings.HasPrefix(string(name), core.ResourceHugePagesPrefix) || strings.HasPrefix(string(name), core.ResourceRequestsHugePagesPrefix)
}

// HugePageResourceName returns a ResourceName with the canonical hugepage
// prefix prepended for the specified page size.  The page size is converted
// to its canonical representation.
func HugePageResourceName(pageSize resource.Quantity) core.ResourceName {
	return core.ResourceName(fmt.Sprintf("%s%s", core.ResourceHugePagesPrefix, pageSize.String()))
}

// HugePageSizeFromResourceName returns the page size for the specified huge page
// resource name.  If the specified input is not a valid huge page resource name
// an error is returned.
func HugePageSizeFromResourceName(name core.ResourceName) (resource.Quantity, error) {
	if !IsHugePageResourceName(name) {
		return resource.Quantity{}, fmt.Errorf("resource name: %s is not valid hugepage name", name)
	}
	pageSize := strings.TrimPrefix(string(name), core.ResourceHugePagesPrefix)
	return resource.ParseQuantity(pageSize)
}

// NonConvertibleFields iterates over the provided map and filters out all but
// any keys with the "non-convertible.kubernetes.io" prefix.
func NonConvertibleFields(annotations map[string]string) map[string]string {
	nonConvertibleKeys := map[string]string{}
	for key, value := range annotations {
		if strings.HasPrefix(key, core.NonConvertibleAnnotationPrefix) {
			nonConvertibleKeys[key] = value
		}
	}
	return nonConvertibleKeys
}

// Semantic can do semantic deep equality checks for core objects.
// Example: apiequality.Semantic.DeepEqual(aPod, aPodWithNonNilButEmptyMaps) == true
var Semantic = conversion.EqualitiesOrDie(
	func(a, b resource.Quantity) bool {
		// Ignore formatting, only care that numeric value stayed the same.
		// TODO: if we decide it's important, it should be safe to start comparing the format.
		//
		// Uninitialized quantities are equivalent to 0 quantities.
		return a.Cmp(b) == 0
	},
	func(a, b metav1.MicroTime) bool {
		return a.UTC() == b.UTC()
	},
	func(a, b metav1.Time) bool {
		return a.UTC() == b.UTC()
	},
	func(a, b labels.Selector) bool {
		return a.String() == b.String()
	},
	func(a, b fields.Selector) bool {
		return a.String() == b.String()
	},
)

var standardResourceQuotaScopes = sets.NewString(
	string(core.ResourceQuotaScopeTerminating),
	string(core.ResourceQuotaScopeNotTerminating),
	string(core.ResourceQuotaScopeBestEffort),
	string(core.ResourceQuotaScopeNotBestEffort),
)

// IsStandardResourceQuotaScope returns true if the scope is a standard value
func IsStandardResourceQuotaScope(str string) bool {
	return standardResourceQuotaScopes.Has(str)
}

var podObjectCountQuotaResources = sets.NewString(
	string(core.ResourcePods),
)

var podComputeQuotaResources = sets.NewString(
	string(core.ResourceCPU),
	string(core.ResourceMemory),
	string(core.ResourceLimitsCPU),
	string(core.ResourceLimitsMemory),
	string(core.ResourceRequestsCPU),
	string(core.ResourceRequestsMemory),
)

// IsResourceQuotaScopeValidForResource returns true if the resource applies to the specified scope
func IsResourceQuotaScopeValidForResource(scope core.ResourceQuotaScope, resource string) bool {
	switch scope {
	case core.ResourceQuotaScopeTerminating, core.ResourceQuotaScopeNotTerminating, core.ResourceQuotaScopeNotBestEffort:
		return podObjectCountQuotaResources.Has(resource) || podComputeQuotaResources.Has(resource)
	case core.ResourceQuotaScopeBestEffort:
		return podObjectCountQuotaResources.Has(resource)
	default:
		return true
	}
}

var standardContainerResources = sets.NewString(
	string(core.ResourceCPU),
	string(core.ResourceMemory),
	string(core.ResourceEphemeralStorage),
)

// IsStandardContainerResourceName returns true if the container can make a resource request
// for the specified resource
func IsStandardContainerResourceName(str string) bool {
	return standardContainerResources.Has(str) || IsHugePageResourceName(core.ResourceName(str))
}

// IsExtendedResourceName returns true if the resource name is not in the
// default namespace.
func IsExtendedResourceName(name core.ResourceName) bool {
	return !IsDefaultNamespaceResource(name)
}

// IsDefaultNamespaceResource returns true if the resource name is in the
// *kubernetes.io/ namespace. Partially-qualified (unprefixed) names are
// implicitly in the kubernetes.io/ namespace.
func IsDefaultNamespaceResource(name core.ResourceName) bool {
	return !strings.Contains(string(name), "/") ||
		strings.Contains(string(name), core.ResourceDefaultNamespacePrefix)
}

var overcommitBlacklist = sets.NewString(string(core.ResourceNvidiaGPU))

// IsOvercommitAllowed returns true if the resource is in the default
// namespace and not blacklisted.
func IsOvercommitAllowed(name core.ResourceName) bool {
	return IsDefaultNamespaceResource(name) &&
		!IsHugePageResourceName(name) &&
		!overcommitBlacklist.Has(string(name))
}

var standardLimitRangeTypes = sets.NewString(
	string(core.LimitTypePod),
	string(core.LimitTypeContainer),
	string(core.LimitTypePersistentVolumeClaim),
)

// IsStandardLimitRangeType returns true if the type is Pod or Container
func IsStandardLimitRangeType(str string) bool {
	return standardLimitRangeTypes.Has(str)
}

var standardQuotaResources = sets.NewString(
	string(core.ResourceCPU),
	string(core.ResourceMemory),
	string(core.ResourceEphemeralStorage),
	string(core.ResourceRequestsCPU),
	string(core.ResourceRequestsMemory),
	string(core.ResourceRequestsStorage),
	string(core.ResourceRequestsEphemeralStorage),
	string(core.ResourceLimitsCPU),
	string(core.ResourceLimitsMemory),
	string(core.ResourceLimitsEphemeralStorage),
	string(core.ResourcePods),
	string(core.ResourceQuotas),
	string(core.ResourceServices),
	string(core.ResourceReplicationControllers),
	string(core.ResourceSecrets),
	string(core.ResourcePersistentVolumeClaims),
	string(core.ResourceConfigMaps),
	string(core.ResourceServicesNodePorts),
	string(core.ResourceServicesLoadBalancers),
)

// IsStandardQuotaResourceName returns true if the resource is known to
// the quota tracking system
func IsStandardQuotaResourceName(str string) bool {
	return standardQuotaResources.Has(str) || IsQuotaHugePageResourceName(core.ResourceName(str))
}

var standardResources = sets.NewString(
	string(core.ResourceCPU),
	string(core.ResourceMemory),
	string(core.ResourceEphemeralStorage),
	string(core.ResourceRequestsCPU),
	string(core.ResourceRequestsMemory),
	string(core.ResourceRequestsEphemeralStorage),
	string(core.ResourceLimitsCPU),
	string(core.ResourceLimitsMemory),
	string(core.ResourceLimitsEphemeralStorage),
	string(core.ResourcePods),
	string(core.ResourceQuotas),
	string(core.ResourceServices),
	string(core.ResourceReplicationControllers),
	string(core.ResourceSecrets),
	string(core.ResourceConfigMaps),
	string(core.ResourcePersistentVolumeClaims),
	string(core.ResourceStorage),
	string(core.ResourceRequestsStorage),
	string(core.ResourceServicesNodePorts),
	string(core.ResourceServicesLoadBalancers),
)

// IsStandardResourceName returns true if the resource is known to the system
func IsStandardResourceName(str string) bool {
	return standardResources.Has(str) || IsQuotaHugePageResourceName(core.ResourceName(str))
}

var integerResources = sets.NewString(
	string(core.ResourcePods),
	string(core.ResourceQuotas),
	string(core.ResourceServices),
	string(core.ResourceReplicationControllers),
	string(core.ResourceSecrets),
	string(core.ResourceConfigMaps),
	string(core.ResourcePersistentVolumeClaims),
	string(core.ResourceServicesNodePorts),
	string(core.ResourceServicesLoadBalancers),
)

// IsIntegerResourceName returns true if the resource is measured in integer values
func IsIntegerResourceName(str string) bool {
	return integerResources.Has(str) || IsExtendedResourceName(core.ResourceName(str))
}

// this function aims to check if the service's ClusterIP is set or not
// the objective is not to perform validation here
func IsServiceIPSet(service *core.Service) bool {
	return service.Spec.ClusterIP != core.ClusterIPNone && service.Spec.ClusterIP != ""
}

var standardFinalizers = sets.NewString(
	string(core.FinalizerKubernetes),
	metav1.FinalizerOrphanDependents,
	metav1.FinalizerDeleteDependents,
)

func IsStandardFinalizerName(str string) bool {
	return standardFinalizers.Has(str)
}

// AddToNodeAddresses appends the NodeAddresses to the passed-by-pointer slice,
// only if they do not already exist
func AddToNodeAddresses(addresses *[]core.NodeAddress, addAddresses ...core.NodeAddress) {
	for _, add := range addAddresses {
		exists := false
		for _, existing := range *addresses {
			if existing.Address == add.Address && existing.Type == add.Type {
				exists = true
				break
			}
		}
		if !exists {
			*addresses = append(*addresses, add)
		}
	}
}

// TODO: make method on LoadBalancerStatus?
func LoadBalancerStatusEqual(l, r *core.LoadBalancerStatus) bool {
	return ingressSliceEqual(l.Ingress, r.Ingress)
}

func ingressSliceEqual(lhs, rhs []core.LoadBalancerIngress) bool {
	if len(lhs) != len(rhs) {
		return false
	}
	for i := range lhs {
		if !ingressEqual(&lhs[i], &rhs[i]) {
			return false
		}
	}
	return true
}

func ingressEqual(lhs, rhs *core.LoadBalancerIngress) bool {
	if lhs.IP != rhs.IP {
		return false
	}
	if lhs.Hostname != rhs.Hostname {
		return false
	}
	return true
}

// TODO: make method on LoadBalancerStatus?
func LoadBalancerStatusDeepCopy(lb *core.LoadBalancerStatus) *core.LoadBalancerStatus {
	c := &core.LoadBalancerStatus{}
	c.Ingress = make([]core.LoadBalancerIngress, len(lb.Ingress))
	for i := range lb.Ingress {
		c.Ingress[i] = lb.Ingress[i]
	}
	return c
}

// GetAccessModesAsString returns a string representation of an array of access modes.
// modes, when present, are always in the same order: RWO,ROX,RWX.
func GetAccessModesAsString(modes []core.PersistentVolumeAccessMode) string {
	modes = removeDuplicateAccessModes(modes)
	modesStr := []string{}
	if containsAccessMode(modes, core.ReadWriteOnce) {
		modesStr = append(modesStr, "RWO")
	}
	if containsAccessMode(modes, core.ReadOnlyMany) {
		modesStr = append(modesStr, "ROX")
	}
	if containsAccessMode(modes, core.ReadWriteMany) {
		modesStr = append(modesStr, "RWX")
	}
	return strings.Join(modesStr, ",")
}

// GetAccessModesAsString returns an array of AccessModes from a string created by GetAccessModesAsString
func GetAccessModesFromString(modes string) []core.PersistentVolumeAccessMode {
	strmodes := strings.Split(modes, ",")
	accessModes := []core.PersistentVolumeAccessMode{}
	for _, s := range strmodes {
		s = strings.Trim(s, " ")
		switch {
		case s == "RWO":
			accessModes = append(accessModes, core.ReadWriteOnce)
		case s == "ROX":
			accessModes = append(accessModes, core.ReadOnlyMany)
		case s == "RWX":
			accessModes = append(accessModes, core.ReadWriteMany)
		}
	}
	return accessModes
}

// removeDuplicateAccessModes returns an array of access modes without any duplicates
func removeDuplicateAccessModes(modes []core.PersistentVolumeAccessMode) []core.PersistentVolumeAccessMode {
	accessModes := []core.PersistentVolumeAccessMode{}
	for _, m := range modes {
		if !containsAccessMode(accessModes, m) {
			accessModes = append(accessModes, m)
		}
	}
	return accessModes
}

func containsAccessMode(modes []core.PersistentVolumeAccessMode, mode core.PersistentVolumeAccessMode) bool {
	for _, m := range modes {
		if m == mode {
			return true
		}
	}
	return false
}

// NodeSelectorRequirementsAsSelector converts the []NodeSelectorRequirement core type into a struct that implements
// labels.Selector.
func NodeSelectorRequirementsAsSelector(nsm []core.NodeSelectorRequirement) (labels.Selector, error) {
	if len(nsm) == 0 {
		return labels.Nothing(), nil
	}
	selector := labels.NewSelector()
	for _, expr := range nsm {
		var op selection.Operator
		switch expr.Operator {
		case core.NodeSelectorOpIn:
			op = selection.In
		case core.NodeSelectorOpNotIn:
			op = selection.NotIn
		case core.NodeSelectorOpExists:
			op = selection.Exists
		case core.NodeSelectorOpDoesNotExist:
			op = selection.DoesNotExist
		case core.NodeSelectorOpGt:
			op = selection.GreaterThan
		case core.NodeSelectorOpLt:
			op = selection.LessThan
		default:
			return nil, fmt.Errorf("%q is not a valid node selector operator", expr.Operator)
		}
		r, err := labels.NewRequirement(expr.Key, op, expr.Values)
		if err != nil {
			return nil, err
		}
		selector = selector.Add(*r)
	}
	return selector, nil
}

// GetTolerationsFromPodAnnotations gets the json serialized tolerations data from Pod.Annotations
// and converts it to the []Toleration type in core.
func GetTolerationsFromPodAnnotations(annotations map[string]string) ([]core.Toleration, error) {
	var tolerations []core.Toleration
	if len(annotations) > 0 && annotations[core.TolerationsAnnotationKey] != "" {
		err := json.Unmarshal([]byte(annotations[core.TolerationsAnnotationKey]), &tolerations)
		if err != nil {
			return tolerations, err
		}
	}
	return tolerations, nil
}

// AddOrUpdateTolerationInPod tries to add a toleration to the pod's toleration list.
// Returns true if something was updated, false otherwise.
func AddOrUpdateTolerationInPod(pod *core.Pod, toleration *core.Toleration) bool {
	podTolerations := pod.Spec.Tolerations

	var newTolerations []core.Toleration
	updated := false
	for i := range podTolerations {
		if toleration.MatchToleration(&podTolerations[i]) {
			if Semantic.DeepEqual(toleration, podTolerations[i]) {
				return false
			}
			newTolerations = append(newTolerations, *toleration)
			updated = true
			continue
		}

		newTolerations = append(newTolerations, podTolerations[i])
	}

	if !updated {
		newTolerations = append(newTolerations, *toleration)
	}

	pod.Spec.Tolerations = newTolerations
	return true
}

// GetTaintsFromNodeAnnotations gets the json serialized taints data from Pod.Annotations
// and converts it to the []Taint type in core.
func GetTaintsFromNodeAnnotations(annotations map[string]string) ([]core.Taint, error) {
	var taints []core.Taint
	if len(annotations) > 0 && annotations[core.TaintsAnnotationKey] != "" {
		err := json.Unmarshal([]byte(annotations[core.TaintsAnnotationKey]), &taints)
		if err != nil {
			return []core.Taint{}, err
		}
	}
	return taints, nil
}

// SysctlsFromPodAnnotations parses the sysctl annotations into a slice of safe Sysctls
// and a slice of unsafe Sysctls. This is only a convenience wrapper around
// SysctlsFromPodAnnotation.
func SysctlsFromPodAnnotations(a map[string]string) ([]core.Sysctl, []core.Sysctl, error) {
	safe, err := SysctlsFromPodAnnotation(a[core.SysctlsPodAnnotationKey])
	if err != nil {
		return nil, nil, err
	}
	unsafe, err := SysctlsFromPodAnnotation(a[core.UnsafeSysctlsPodAnnotationKey])
	if err != nil {
		return nil, nil, err
	}

	return safe, unsafe, nil
}

// SysctlsFromPodAnnotation parses an annotation value into a slice of Sysctls.
func SysctlsFromPodAnnotation(annotation string) ([]core.Sysctl, error) {
	if len(annotation) == 0 {
		return nil, nil
	}

	kvs := strings.Split(annotation, ",")
	sysctls := make([]core.Sysctl, len(kvs))
	for i, kv := range kvs {
		cs := strings.Split(kv, "=")
		if len(cs) != 2 || len(cs[0]) == 0 {
			return nil, fmt.Errorf("sysctl %q not of the format sysctl_name=value", kv)
		}
		sysctls[i].Name = cs[0]
		sysctls[i].Value = cs[1]
	}
	return sysctls, nil
}

// PodAnnotationsFromSysctls creates an annotation value for a slice of Sysctls.
func PodAnnotationsFromSysctls(sysctls []core.Sysctl) string {
	if len(sysctls) == 0 {
		return ""
	}

	kvs := make([]string, len(sysctls))
	for i := range sysctls {
		kvs[i] = fmt.Sprintf("%s=%s", sysctls[i].Name, sysctls[i].Value)
	}
	return strings.Join(kvs, ",")
}

// GetPersistentVolumeClass returns StorageClassName.
func GetPersistentVolumeClass(volume *core.PersistentVolume) string {
	// Use beta annotation first
	if class, found := volume.Annotations[core.BetaStorageClassAnnotation]; found {
		return class
	}

	return volume.Spec.StorageClassName
}

// GetPersistentVolumeClaimClass returns StorageClassName. If no storage class was
// requested, it returns "".
func GetPersistentVolumeClaimClass(claim *core.PersistentVolumeClaim) string {
	// Use beta annotation first
	if class, found := claim.Annotations[core.BetaStorageClassAnnotation]; found {
		return class
	}

	if claim.Spec.StorageClassName != nil {
		return *claim.Spec.StorageClassName
	}

	return ""
}

// PersistentVolumeClaimHasClass returns true if given claim has set StorageClassName field.
func PersistentVolumeClaimHasClass(claim *core.PersistentVolumeClaim) bool {
	// Use beta annotation first
	if _, found := claim.Annotations[core.BetaStorageClassAnnotation]; found {
		return true
	}

	if claim.Spec.StorageClassName != nil {
		return true
	}

	return false
}

// GetStorageNodeAffinityFromAnnotation gets the json serialized data from PersistentVolume.Annotations
// and converts it to the NodeAffinity type in core.
// TODO: update when storage node affinity graduates to beta
func GetStorageNodeAffinityFromAnnotation(annotations map[string]string) (*core.NodeAffinity, error) {
	if len(annotations) > 0 && annotations[core.AlphaStorageNodeAffinityAnnotation] != "" {
		var affinity core.NodeAffinity
		err := json.Unmarshal([]byte(annotations[core.AlphaStorageNodeAffinityAnnotation]), &affinity)
		if err != nil {
			return nil, err
		}
		return &affinity, nil
	}
	return nil, nil
}

// Converts NodeAffinity type to Alpha annotation for use in PersistentVolumes
// TODO: update when storage node affinity graduates to beta
func StorageNodeAffinityToAlphaAnnotation(annotations map[string]string, affinity *core.NodeAffinity) error {
	if affinity == nil {
		return nil
	}

	json, err := json.Marshal(*affinity)
	if err != nil {
		return err
	}
	annotations[core.AlphaStorageNodeAffinityAnnotation] = string(json)
	return nil
}
