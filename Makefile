all: docker

.PHONY: docker
docker: build
    # sudo docker build -t "xujieasd/enn-policy" .

.PHONY: build
build:
	@echo "enn-policy binary build Starting."
	CGO_ENABLED=0 go build -o enn-policy enn-policy.go
	@echo "enn-policy binary build finished."

.PHONY: test
test:
	@echo "enn-policy unit test Starting."
	hack/test.sh
	@echo "enn-policy unit test finished."

.PHONY: clean
clean:
	rm -f enn-policy