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

package history

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	hashutil "octorun.github.io/octorun/util/hash"
)

// RevisionHash hashes the contents of revision's Data using FNV hashing. If probe is not nil, the byte value
// of probe is added written to the hash as well. The returned hash will be a safe encoded string to avoid bad words.
func RevisionHash(revision *appsv1.ControllerRevision, probe *int32) string {
	hf := fnv.New32()
	if len(revision.Data.Raw) > 0 {
		_, _ = hf.Write(revision.Data.Raw)
	}

	if revision.Data.Object != nil {
		hashutil.DeepHashObject(hf, revision.Data.Object)
	}

	if probe != nil {
		_, _ = hf.Write([]byte(strconv.FormatInt(int64(*probe), 10)))
	}

	return rand.SafeEncodeString(fmt.Sprint(hf.Sum32()))
}

// func ObjectRevisions()

// func ObjectRevision(obj runtime.Object, objTemplate runtime.Object, scheme runtime.Scheme, codec runtime.Codec) (*appsv1.ControllerRevision, error) {
// 	templateLabels := objTemplate.(metav1.ObjectMetaAccessor).GetObjectMeta().GetLabels()
// 	revisionLabels := make(map[string]string)
// 	for k, v := range templateLabels {
// 		revisionLabels[k] = v
// 	}

// 	cr := &appsv1.ControllerRevision{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Labels: revisionLabels,
// 		},
// 		Data:     runtime.RawExtension{Raw: rawData},
// 		Revision: revision,
// 	}

// 	return nil, nil
// }

// ObjectFromRevision restore the given object from state in revision.
func ObjectFromRevision(obj runtime.Object, codec runtime.Codec, revision *appsv1.ControllerRevision) error {
	copiedObj := obj.DeepCopyObject()
	orig := runtime.EncodeOrDie(codec, copiedObj)
	patched, err := strategicpatch.StrategicMergePatch([]byte(orig), revision.Data.Raw, copiedObj)
	if err != nil {
		return err
	}

	return json.Unmarshal(patched, obj)
}

// ObjectPatch returns a strategic merge patch for given object that can be applied to restore a Object to a
// previous version.
func ObjectPatch(obj runtime.Object, codec runtime.Codec) ([]byte, error) {
	data, err := runtime.Encode(codec, obj)
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	objCopy := make(map[string]interface{})
	specCopy := make(map[string]interface{})

	spec := raw["spec"].(map[string]interface{})
	template := spec["template"].(map[string]interface{})
	specCopy["template"] = template
	template["$patch"] = "replace"

	objCopy["spec"] = specCopy
	return json.Marshal(objCopy)
}
