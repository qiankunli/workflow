package operators

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/qiankunli/workflow/pkg/apis/workflow/v1alpha1"
)

func (r *workflowReconciler) onStart(ctx context.Context, workflow *v1alpha1.Workflow) error {
	err := r.callCallback(workflow)
	return err
}

func (r *workflowReconciler) onChange(ctx context.Context, workflow *v1alpha1.Workflow) error {
	err := r.callCallback(workflow)
	return err
}

func (r *workflowReconciler) onSuccess(ctx context.Context, workflow *v1alpha1.Workflow) error {
	err := r.callCallback(workflow)
	return err
}

func (r *workflowReconciler) onRollback(ctx context.Context, workflow *v1alpha1.Workflow) error {
	err := r.callCallback(workflow)
	return err
}

func (r *workflowReconciler) onDeleted(ctx context.Context, workflow *v1alpha1.Workflow) error {
	err := r.callCallback(workflow)
	return err
}

func (r *workflowReconciler) callCallback(workflow *v1alpha1.Workflow) error {
	log := r.log.WithValues("name", workflow.Name)
	data := map[string]interface{}{
		"name":          workflow.Name,
		"phase":         workflow.Status.Phase,
		"attributes":    workflow.Status.Attributes,
		"runError":      workflow.Status.RunError,
		"rollbackError": workflow.Status.RollbackError,
		"syncError":     workflow.Status.SyncError,
	}
	// 将数据编码为 JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	// 发出 POST 请求
	resp, err := http.Post(workflow.Spec.Callback.Url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()
	// 根据状态码处理响应
	switch resp.StatusCode {
	case http.StatusOK:
		log.V(4).Info("workflow callback url success", "url", workflow.Spec.Callback.Url)
		return nil
	case http.StatusNotFound:
		log.V(4).Info("workflow callback url 404", "url", workflow.Spec.Callback.Url)
		if workflow.Spec.Callback.IgnoreNotFound {
			return nil
		}
		return fmt.Errorf("workflow callback url %s 404", workflow.Spec.Callback.Url)
	default:
		errorMsg := ""
		if body, err := ioutil.ReadAll(resp.Body); err == nil {
			errorMsg = string(body)
		}
		log.V(2).Info("workflow callback url error", "url", workflow.Spec.Callback.Url, "statusCode",
			resp.StatusCode, "err", errorMsg)
		return fmt.Errorf("workflow callback url %s error %s", workflow.Spec.Callback.Url, errorMsg)
	}
}
