image:
	go mod vendor
	docker build -t otic-test-app:dev .
	rm -rf vendor

apply:
	kubectl apply -f deploy/deployment.yaml

delete:
	kubectl delete -f deploy/deployment.yaml

kpm-test:
	kubectl exec -n sdran -it $(shell kubectl get pod -n sdran -l app=otic-test-app -o name) -- kpm.test -test.v

rc-test:
	kubectl exec -n sdran -it $(shell kubectl get pod -n sdran -l app=otic-test-app -o name) -- rc.test -test.v

upgrade: delete image apply
