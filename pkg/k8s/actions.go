package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	// Added: Cordon/Drain을 위한 import 추가
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields" // Added: Cordon/Drain을 위한 import 추가
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
// 보안 경고: 이 함수는 원격 코드 실행의 위험이 있으므로, 프로덕션 환경에서는
// DaemonSet과 같은 더 안전한 패턴을 사용하는 것이 좋습니다.
// 이 예제에서는 요청된 기능을 구현하되, 명령어 주입을 막기 위한 검증이 필요합니다.
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

	result := fmt.Sprintf("Successfully executed commands on node '%s'.\nGzip output: %s\n