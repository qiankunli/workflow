package kube

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	corev1 "k8s.io/api/core/v1"
	k8sapierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ErrSkip = fmt.Errorf("skip")

func RetryUpdateOnConflict(ctx context.Context, cli client.Client, obj client.Object, mutateFn func() error) (err error) {
	start := time.Now()
	retryCount := -1
	diff := true
	defer func() {
		duration := time.Since(start)
		klog.V(4).Infof("RetryUpdateOnConflict: %v, duration: %v, retry: %v, diff: %v, err: %v", client.ObjectKeyFromObject(obj).String(), duration.String(), retryCount, diff, err)
	}()
	return retry.RetryOnConflict(retry.DefaultRetry, func() (err error) {
		retryCount++
		key := client.ObjectKeyFromObject(obj)
		err = cli.Get(ctx, key, obj)
		if err != nil {
			return
		}
		// 缓存里的最新值
		base := obj.DeepCopyObject()
		err = mutateFn()
		if err != nil {
			if err == ErrSkip {
				err = nil
			}
			return
		}
		// check diff
		rawBase, _ := json.Marshal(base)
		rawObj, _ := json.Marshal(obj)
		retBase := trimJSONObject(rawBase)
		retObj := trimJSONObject(rawObj)
		if string(retBase) == string(retObj) {
			diff = false
			return
		}
		err = cli.Update(ctx, obj)
		return
	})
}

func trimJSONObject(src []byte) (dst []byte) {
	dst, _ = sjson.DeleteBytes(src, "status")
	dst, _ = sjson.DeleteBytes(dst, "metadata.resourceVersion")
	dst, _ = sjson.DeleteBytes(dst, "metadata.generation")
	dst, _ = sjson.DeleteBytes(dst, "metadata.managedFields")
	dst, _ = sjson.DeleteBytes(dst, "metadata.creationTimestamp")
	dst, _ = sjson.DeleteBytes(dst, "metadata.uid")
	return
}

func RetryUpdateStatusOnConflict(ctx context.Context, cli client.Client, obj client.Object, mutateFn func() error) (err error) {
	start := time.Now()
	retryCount := -1
	diff := true
	defer func() {
		duration := time.Since(start)
		klog.V(4).Infof("RetryUpdateStatusOnConflict: %v, duration: %v, retry: %v, diff: %v, err: %v", client.ObjectKeyFromObject(obj).String(), duration.String(), retryCount, diff, err)
	}()
	return retry.RetryOnConflict(retry.DefaultRetry, func() (err error) {
		retryCount++
		key := client.ObjectKeyFromObject(obj)
		err = cli.Get(ctx, key, obj)
		if err != nil {
			return
		}
		base := obj.DeepCopyObject()
		err = mutateFn()
		if err != nil {
			return
		}
		// check diff
		rawBase, _ := json.Marshal(base)
		rawObj, _ := json.Marshal(obj)
		retBase := gjson.GetBytes(rawBase, "status")
		retObj := gjson.GetBytes(rawObj, "status")
		if retBase.String() == retObj.String() {
			diff = false
			return
		}
		err = cli.Status().Update(ctx, obj)
		return
	})
}

// GetSecret returns value of key in secret
func GetSecret(client kubernetes.Interface, namespace string, secretRef *corev1.SecretKeySelector) (string, error) {
	if secretRef == nil {
		return "", errors.New("empty secret ref")
	}

	secret, err := client.CoreV1().Secrets(namespace).Get(context.TODO(), secretRef.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get secret: %w", err)
	}
	value, ok := secret.Data[secretRef.Key]
	if !ok {
		return "", fmt.Errorf("empty key %s in secret %s", secretRef.Key, secretRef.Name)
	}
	return string(value), nil
}

// TryGetConfigMap tries several times to get configmap
func TryGetConfigMap(client kubernetes.Interface, namespace string, configmapRef *corev1.ConfigMapKeySelector) (string, error) {
	if configmapRef == nil {
		return "", errors.New("empty configmap ref")
	}

	for i := 0; i < 5; i++ {
		value, err := GetConfigMap(client, namespace, configmapRef)
		if err == nil {
			return value, nil
		}
		klog.ErrorS(err, "wait to get configmap")
		time.Sleep(time.Second)
	}

	return "", errors.New("timeout waiting to get configmap")
}

// GetConfigMap returns value of key in configmap
func GetConfigMap(client kubernetes.Interface, namespace string, configmapRef *corev1.ConfigMapKeySelector) (string, error) {
	if configmapRef == nil {
		return "", errors.New("empty configmap ref")
	}

	configmap, err := client.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configmapRef.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get configmap: %w", err)
	}
	value, ok := configmap.Data[configmapRef.Key]
	if !ok {
		return "", fmt.Errorf("empty key %s in configmap %s", configmapRef.Key, configmapRef.Name)
	}
	return value, nil
}

// GetSecretV2 returns value of key in secret by client.Client
func GetSecretV2(ctx context.Context, c client.Client, namespace string, secretRef *corev1.SecretKeySelector) ([]byte, error) {
	if secretRef == nil {
		return nil, errors.New("empty secret ref")
	}

	secret := &corev1.Secret{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: secretRef.Name}, secret); err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}
	value, ok := secret.Data[secretRef.Key]
	if !ok {
		return nil, fmt.Errorf("empty key %s in secret %s", secretRef.Key, secretRef.Name)
	}
	return value, nil
}

// CreateOrUpdateSecretV2 create or update secret by client.Client
func CreateOrUpdateSecretV2(ctx context.Context, c client.Client, namespace string, secretType corev1.SecretType, secretRef *corev1.SecretKeySelector, data []byte, owners ...client.Object) error {
	oldSecret := &corev1.Secret{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: secretRef.Name}, oldSecret); err != nil {
		if !k8sapierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get secret: %w", err)
		}
		ownerRefs := make([]metav1.OwnerReference, len(owners))
		for i := range owners {
			if namespace != owners[i].GetNamespace() {
				return fmt.Errorf("owner namespace '%s' no equal to secret namespace '%s'", owners[i].GetNamespace(), namespace)
			}
			ownerRefs[i] = *metav1.NewControllerRef(owners[i], owners[i].GetObjectKind().GroupVersionKind())
		}
		// not exist, create it
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:       namespace,
				Name:            secretRef.Name,
				OwnerReferences: ownerRefs,
			},
			Type: secretType,
			Data: map[string][]byte{
				secretRef.Key: data,
			},
		}
		if err = c.Create(ctx, secret); err != nil {
			return fmt.Errorf("failed to create secret: %w", err)
		}
		return nil
	}

	if bytes.Equal(oldSecret.Data[secretRef.Key], data) {
		return nil
	}

	oldSecret.Data[secretRef.Key] = data
	if err := c.Update(ctx, oldSecret); err != nil {
		return fmt.Errorf("failed to update secret: %w", err)
	}
	return nil
}

// DeleteSecretV2 delete secret by client.Client
func DeleteSecretV2(ctx context.Context, c client.Client, namespace, name string) error {
	if err := c.Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name}}); err != nil {
		if !k8sapierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete secret: %w", err)
		}
	}
	return nil
}
