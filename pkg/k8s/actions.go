package k8s

import (
    "context"
    "encoding/json"
    "fmt"
    "os/exec"
    "time"

    corev1 "k8s.io/api/core/v1"
    policyv1beta1 "k8s.io/api/policy/v1beta1"
    apierrors "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/fields"
    "k8s.io/apimachinery/pkg/types"
    "k8s.io/client-go/kubernetes"
)

// RestartDeployment는 kubectl rollout restart와 동일한 효과를 내기 위해
// Deployment 템플릿에 어노테이션을 패치합니다.
func RestartDeployment(clientset *kubernetes.Clientset, namespace, deploymentName string) error {
    patchData := map[string]interface{}{
        "spec": map[string]interface{}{
            "template": map[string]interface{}{
                "metadata": map[string]interface{}{
                    "annotations": map[string]string{
                        "kubectl.kubernetes.io/restartedAt": time.Now().Format(time.RFC3339),
                    },
                },
            },
        },
    }

    patchBytes, err := json.Marshal(patchData)
    if err != nil {
        return fmt.Errorf("failed to marshal patch data: %w", err)
    }

    _, err = clientset.AppsV1().Deployments(namespace).Patch(
        context.TODO(),
        deploymentName,
        types.StrategicMergePatchType,
        patchBytes,
        metav1.PatchOptions{},
    )
    if err != nil {
        return fmt.Errorf("failed to patch deployment: %w", err)
    }

    return nil
}

// ExecuteNodeShellCommand는 노드에서 셸 명령을 실행합니다.
func ExecuteNodeShellCommand(clientset *kubernetes.Clientset, nodeName string, parameters map[string]string) (string, error) {
    logPath, ok := parameters["logPath"]
    if !ok || logPath == "" {
        return "", fmt.Errorf("logPath parameter is missing or empty")
    }

    // 1단계: gzip 압축
    gzipCmd := exec.Command("gzip", logPath)
    gzipOutput, err := gzipCmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("failed to gzip log file '%s': %s, error: %w",
            logPath, string(gzipOutput), err)
    }

    // 2단계: logrotate 강제 실행
    logrotateCmd := exec.Command("logrotate", "-f", "/etc/logrotate.conf")
    logrotateOutput, err := logrotateCmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("failed to force logrotate: %s, error: %w",
            string(logrotateOutput), err)
    }

    result := fmt.Sprintf(
        "Successfully executed commands on node '%s'.\nGzip output: %s\nLogrotate output: %s",
        nodeName, string(gzipOutput), string(logrotateOutput),
    )

    return result, nil
}

// CordonNode는 노드를 스케줄 불가능(unschedulable) 상태로 만듭니다.
func CordonNode(clientset *kubernetes.Clientset, nodeName string) error {
    patchData := []byte(`{"spec":{"unschedulable":true}}`)

    _, err := clientset.CoreV1().Nodes().Patch(
        context.TODO(),
        nodeName,
        types.StrategicMergePatchType,
        patchData,
        metav1.PatchOptions{},
    )
    if err != nil {
        return fmt.Errorf("failed to cordon node '%s': %w", nodeName, err)
    }
    return nil
}

// DrainNode는 DaemonSet을 제외한 Pod를 Eviction API를 통해 제거합니다.
func DrainNode(clientset *kubernetes.Clientset, nodeName string) error {
    // 1. 먼저 노드를 Cordon 처리
    if err := CordonNode(clientset, nodeName); err != nil {
        return err
    }

    // 2. 노드에서 실행 중인 모든 Pod 목록 조회
    fieldSelector := fields.SelectorFromSet(fields.Set{"spec.nodeName": nodeName})
    podList, err := clientset.CoreV1().Pods(corev1.NamespaceAll).List(
        context.TODO(),
        metav1.ListOptions{
            FieldSelector: fieldSelector.String(),
        },
    )
    if err != nil {
        return fmt.Errorf("failed to list pods on node '%s': %w", nodeName, err)
    }

    // 3. 각 Pod를 순회하며 Evict
    for _, pod := range podList.Items {
        // 3a. DaemonSet Pod는 건너뜀
        isDaemonSetPod := false
        for _, ownerRef := range pod.OwnerReferences {
            if ownerRef.Kind == "DaemonSet" {
                isDaemonSetPod = true
                break
            }
        }
        if isDaemonSetPod {
            continue
        }

        // 3b. Eviction API 사용
        eviction := &policyv1beta1.Eviction{
            ObjectMeta: metav1.ObjectMeta{
                Name:      pod.Name,
                Namespace: pod.Namespace,
            },
            DeleteOptions: &metav1.DeleteOptions{},
        }

        err := clientset.CoreV1().Pods(pod.Namespace).Evict(
            context.TODO(),
            eviction,
        )
        if err != nil {
            if apierrors.IsNotFound(err) || apierrors.IsTooManyRequests(err) {
                // 이미 사라졌거나, PDB 때문에 막힌 경우는 무시
                continue
            }
            return fmt.Errorf("failed to evict pod '%s/%s': %w", pod.Namespace, pod.Name, err)
        }
    }

    return nil
}
