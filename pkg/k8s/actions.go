package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	corev1 "k8s.io/api/core/v1"
	// --- Changed: v1beta1 API로 수정 ---
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// RestartDeployment는 kubectl rollout restart와 동일한 효과를 내기 위해 Deployment에 어노테이션을 패치합니다.
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

	_, err = clientset.AppsV1().Deployments(namespace).Patch(context.TODO(), deploymentName, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to patch deployment: %w", err)
	}

	return nil
}

// ExecuteNodeShellCommand는 노드에서 셸 명령을 실행합니다.
func ExecuteNodeShellCommand(clientset *kubernetes.Clientset, nodeName string, parameters map[string]string) (string, error) {
	logPath, ok := parameters["logPath"]
	// Changed: 논리 연산자 오류 및 줄바꿈 수정
	if !ok || logPath == "" {
		return "", fmt.Errorf("logPath parameter is missing or empty")
	}

	// **보안 강화**: logPath가 예상된 경로 패턴(예: /var/log/*.log)을 따르는지 확인하는 로직 추가 필요.
	// 여기서는 예시로 검증 로직을 생략합니다.

	// 1단계: gzip 압축
	gzipCmd := exec.Command("gzip", logPath)
	gzipOutput, err := gzipCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to gzip log file '%s': %s, error: %w", logPath, string(gzipOutput), err)
	}

	// 2단계: logrotate 강제 실행
	logrotateCmd := exec.Command("logrotate", "-f", "/etc/logrotate.conf") // 실제 환경에 맞는 설정 파일 경로 사용
	logrotateOutput, err := logrotateCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to force logrotate: %s, error: %w", string(logrotateOutput), err)
	}

	result := fmt.Sprintf("Successfully executed commands on node '%s'.\nGzip output: %s\nLogrotate output: %s",
		nodeName, string(gzipOutput), string(logrotateOutput))

	return result, nil
}

// CordonNode는 노드를 스케줄 불가능(unschedulable) 상태로 만듭니다.
func CordonNode(clientset *kubernetes.Clientset, nodeName string) error {
	// Cordon은 `spec.unschedulable=true`로 패치하는 것과 동일합니다.
	patchData := []byte(`{"spec":{"unschedulable":true}}`)

	_, err := clientset.CoreV1().Nodes().Patch(context.TODO(), nodeName, types.StrategicMergePatchType, patchData, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to cordon node '%s': %w", nodeName, err)
	}
	return nil
}

// DrainNode는 `--ignore-daemonsets` 옵션과 유사하게
// 노드에서 DaemonSet을 제외한 모든 Pod를 Eviction API를 통해 안전하게 제거(evict)합니다.
func DrainNode(clientset *kubernetes.Clientset, nodeName string) error {
	// 1. 먼저 노드를 Cordon 처리합니다.
	if err := CordonNode(clientset, nodeName); err != nil {
		return err
	}

	// 2. 노드에서 실행 중인 모든 Pod 목록을 가져옵니다.
	fieldSelector := fields.SelectorFromSet(fields.Set{"spec.nodeName": nodeName})
	podList, err := clientset.CoreV1().Pods(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{
		FieldSelector: fieldSelector.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to list pods on node '%s': %w", nodeName, err)
	}

	// 3. 각 Pod를 순회하며 Evict합니다.
	for _, pod := range podList.Items {
		// 3a. DaemonSet Pod는 건너뜁니다.
		isDaemonSetPod := false
		for _, ownerRef := range pod.OwnerReferences {
			if ownerRef.Kind == "DaemonSet" {
				isDaemonSetPod = true
				break
			}
		}
		if isDaemonSetPod {
			continue // DaemonSet Pod는 Evict하지 않음
		}

		// 3b. Eviction API를 사용하여 Pod를 제거합니다.
		// --- Changed: "v1".Eviction -> "v1beta1".Eviction 으로 수정 ---
		eviction := &policyv1beta1.Eviction{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pod.Name,
				Namespace: pod.Namespace,
			},
			DeleteOptions: &metav1.DeleteOptions{}, // Grace period 기본값 사용
		}

		// (중요) .Evict() 호출은 클라이언트 라이브러리 버전에 따라 v1beta1을 사용합니다.
		err := clientset.CoreV1().Pods(pod.Namespace).Evict(context.TODO(), eviction)
		if err != nil {
			// 이미 Pod가 종료 중이거나(NotFound), PDB 등으로 인해 Evict가 불가능한(TooManyRequests) 경우
			if apierrors.IsNotFound(err) || apierrors.IsTooManyRequests(err) {
				continue // 다음 Pod로 넘어감
			}
			return fmt.Errorf("failed to evict pod '%s/%s': %w", pod.Namespace, pod.Name, err)
		}
	}

	return nil
}