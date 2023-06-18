package kube

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func unstructuredHasStatus(u *unstructured.Unstructured) bool {
	_, ok := u.Object["status"]
	return ok
}

// ToUnstructured converts an object into map[string]interface{} representation
func ToUnstructured(obj runtime.Object) (*unstructured.Unstructured, error) {
	if _, ok := obj.(runtime.Unstructured); ok {
		obj = obj.DeepCopyObject()
	}
	rawMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: rawMap}, nil
}

func unsafeUnstructuredCopy(obj *unstructured.Unstructured) *unstructured.Unstructured {
	res := &unstructured.Unstructured{
		Object: make(map[string]interface{}, len(obj.Object)),
	}

	for key := range obj.Object {
		value := obj.Object[key]
		res.Object[key] = value
	}
	return res
}
