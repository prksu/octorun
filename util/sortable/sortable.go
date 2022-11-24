/*
Copyright 2022 The Authors.

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

package sortable

import (
	appsv1 "k8s.io/api/apps/v1"

	octorunv1 "octorun.github.io/octorun/api/v1alpha2"
)

const (
	runnerMustDelete    float64 = 100.0
	runnerCouldDelete   float64 = 50.0
	runnermustNotDelete float64 = 0.0
)

// RunnersToDelete is sortable slice of Runner
// implement sort.Interface.
type RunnersToDelete []*octorunv1.Runner

func (r RunnersToDelete) Len() int      { return len(r) }
func (r RunnersToDelete) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r RunnersToDelete) Less(i, j int) bool {
	priority := func(runner *octorunv1.Runner) float64 {
		if !runner.GetDeletionTimestamp().IsZero() {
			return runnerMustDelete
		}

		if runner.Status.Phase == octorunv1.RunnerActivePhase {
			return runnermustNotDelete
		}

		return runnerCouldDelete
	}

	return priority(r[j]) < priority(r[i])
}

// Revisions is sortable slice of ControllerRevision
// implement sort.Interface.
type Revisions []*appsv1.ControllerRevision

func (r Revisions) Len() int      { return len(r) }
func (r Revisions) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r Revisions) Less(i, j int) bool {
	if r[i].Revision == r[j].Revision {
		if r[j].CreationTimestamp.Equal(&r[i].CreationTimestamp) {
			return r[i].Name < r[j].Name
		}
		return r[j].CreationTimestamp.After(r[i].CreationTimestamp.Time)
	}

	return r[i].Revision < r[j].Revision
}
