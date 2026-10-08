MODULES := $(shell find . -maxdepth 2 -name go.mod | xargs -I{} dirname {} | sort)

.PHONY: build test vet tidy fix clean

build:
	@for m in $(MODULES); do \
		bin=$$(basename $$m); \
		echo "==> $$m (-> $$m/$$bin)"; \
		(cd $$m && go build -o $$bin .) || exit 1; \
	done

clean:
	@for m in $(MODULES); do \
		bin=$$(basename $$m); \
		rm -f $$m/$$bin; \
	done
	@echo "removed per-sample binaries"

test:
	@for m in $(MODULES); do \
		echo "==> $$m"; \
		(cd $$m && go test ./...) || exit 1; \
	done

vet:
	@for m in $(MODULES); do \
		echo "==> $$m"; \
		(cd $$m && go vet ./...) || exit 1; \
	done

fix:
	@for m in $(MODULES); do \
		echo "==> $$m"; \
		(cd $$m && go fix ./...) || exit 1; \
	done

tidy:
	@for m in $(MODULES); do \
		echo "==> $$m"; \
		(cd $$m && go mod tidy) || exit 1; \
	done
