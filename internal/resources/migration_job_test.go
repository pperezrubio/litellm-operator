/*
Copyright 2026 bitkaio LLC.

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

package resources

import (
	"testing"

	litellmv1alpha1 "github.com/PalenaAI/litellm-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func TestBuildMigrationJob_RestrictedSecurityContext(t *testing.T) {
	instance := newTestInstance()
	nonRoot := true
	instance.Spec.Security = &litellmv1alpha1.SecuritySpec{RunAsNonRoot: &nonRoot}
	labels := map[string]string{"app": "litellm"}

	job := BuildMigrationJob(instance, labels)

	podSC := job.Spec.Template.Spec.SecurityContext
	if podSC == nil || podSC.SeccompProfile == nil || podSC.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
		t.Fatalf("expected pod seccompProfile RuntimeDefault, got %#v", podSC)
	}
	if podSC.RunAsNonRoot == nil || !*podSC.RunAsNonRoot {
		t.Fatalf("expected pod runAsNonRoot=true, got %#v", podSC.RunAsNonRoot)
	}

	containerSC := job.Spec.Template.Spec.Containers[0].SecurityContext
	if containerSC == nil {
		t.Fatal("expected container securityContext")
	}
	if containerSC.AllowPrivilegeEscalation == nil || *containerSC.AllowPrivilegeEscalation {
		t.Fatalf("expected allowPrivilegeEscalation=false, got %#v", containerSC.AllowPrivilegeEscalation)
	}
	if containerSC.Capabilities == nil || len(containerSC.Capabilities.Drop) != 1 || containerSC.Capabilities.Drop[0] != corev1.Capability("ALL") {
		t.Fatalf("expected capabilities.drop=[ALL], got %#v", containerSC.Capabilities)
	}
	if containerSC.RunAsNonRoot == nil || !*containerSC.RunAsNonRoot {
		t.Fatalf("expected container runAsNonRoot=true, got %#v", containerSC.RunAsNonRoot)
	}
}
