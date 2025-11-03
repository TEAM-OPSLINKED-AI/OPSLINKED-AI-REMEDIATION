$ ./opslinked-ai-remediation
{"level":"info","ts":"2025-10-06T19:54:43.646+0900","caller":"OPSLINKED-AI-REMEDIATION/main.go:32","msg":"Starting Remediation Module","logLevel":"info"}
{"level":"info","ts":"2025-10-06T19:54:43.745+0900","caller":"OPSLINKED-AI-REMEDIATION/main.go:39","msg":"Successfully initialized Kubernetes client"}
{"level":"info","ts":"2025-10-06T19:54:43.745+0900","caller":"OPSLINKED-AI-REMEDIATION/main.go:58","msg":"HTTP server starting","port":8080}


[root@kubernetes-host ~]# curl -X POST http://172.20.112.101:32563/remediate \
-H "Content-Type: application/json" \
-d '{
    "actionType": "RESTART_DEPLOYMENT",
    "namespace": "default",
    "resourceName": "wwwm-spring-dummy",
    "reason": "Simulated JVM Heap Pressure (jvm_memory_used > 0.9)",
    "triggeredBy": "Simulation-CURL"
}'
Remediation action accepted for asynchronous processing.

[root@master opslinked-ai]# k logs -f remediation-module-5779b4757d-vr7x7
{"level":"info","ts":"2025-11-03T10:29:20.048Z","caller":"handlers/remediation.go:68","msg":"Starting to handle remediation action","actionType":"RESTART_DEPLOYMENT","namespace":"default","resourceName":"wwwm-spring-dummy"}
{"level":"info","ts":"2025-11-03T10:29:20.074Z","caller":"handlers/remediation.go:96","msg":"Action execution successful","actionType":"RESTART_DEPLOYMENT","namespace":"default","resourceName":"wwwm-spring-dummy","details":"Successfully triggered rolling restart for Deployment 'default/wwwm-spring-dummy'.","duration":0.02646792}

[root@kubernetes-host ~]# curl -X POST http://172.20.112.101:32563/remediate \
-H "Content-Type: application/json" \
-d '{
    "actionType": "EXECUTE_NODE_SHELL_COMMAND",
    "resourceName": "k8s-worker-1",
    "parameters": {
      "logPath": "/var/log/test-app/app.log"
    },
    "reason": "Simulated Log Rotation Test (Success)",
    "triggeredBy": "Simulation-CURL"
}'
Remediation action accepted for asynchronous processing.

[root@master opslinked-ai]# k logs -f remediation-module-5779b4757d-vr7x7
NODE_SHELL_COMMAND","namespace":"","resourceName":"k8s-worker-1","details":"Successfully executed commands on node 'k8s-worker-1'.\nGzip output: \nLogrotate output: ","duration":0.006278146}


[root@kubernetes-host ~]# curl -X POST http://172.20.112.101:32563/remediate \
-H "Content-Type: application/json" \
-d '{
    "actionType": "RESTART_DEPLOYMENT",
    "namespace": "default",
    "resourceName": "wwwm-mysql-dummy",
    "reason": "Simulated DB Connection Pool Exhaustion (threads_connected > 95%)",
    "triggeredBy": "Simulation-CURL"
}'
Remediation action accepted for asynchronous processing.

[root@master opslinked-ai]# k logs -f remediation-module-5779b4757d-vr7x7
{"level":"info","ts":"2025-11-03T10:29:37.756Z","caller":"handlers/remediation.go:68","msg":"Starting to handle remediation action","actionType":"RESTART_DEPLOYMENT","namespace":"default","resourceName":"wwwm-mysql-dummy"}
{"level":"info","ts":"2025-11-03T10:29:37.777Z","caller":"handlers/remediation.go:96","msg":"Action execution successful","actionType":"RESTART_DEPLOYMENT","namespace":"default","resourceName":"wwwm-mysql-dummy","details":"Successfully triggered rolling restart for Deployment 'default/wwwm-mysql-dummy'.","duration":0.020674729}

[root@master opslinked-ai]# k get pod -w
wwwm-spring-dummy-55cbc4b657-lnv4t               1/1     Terminating   0          12m
wwwm-spring-dummy-6465d6fdbb-dbqh8               1/1     Running       0          8s
wwwm-mysql-dummy-84dbf6fd57-dl4vz                0/1     Pending       0          0s
wwwm-mysql-dummy-84dbf6fd57-dl4vz                0/1     Pending       0          0s
wwwm-mysql-dummy-84dbf6fd57-dl4vz                0/1     ContainerCreating   0          0s
wwwm-mysql-dummy-84dbf6fd57-dl4vz                1/1     Running             0          1s
wwwm-mysql-dummy-f7685bcbb-ghvck                 1/1     Terminating         0          12m
wwwm-spring-dummy-55cbc4b657-lnv4t               0/1     Error               0          12m
wwwm-spring-dummy-55cbc4b657-lnv4t               0/1     Error               0          12m
wwwm-spring-dummy-55cbc4b657-lnv4t               0/1     Error               0          12