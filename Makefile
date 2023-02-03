local_package=gorm.io/gormx
all: lint build test

check:
	@$(set_env) ./.github/pre_install.sh
	@$(set_env) test -z "$$(goimports -local $(local_package) -d .)"
	@$(set_env) test -z "$$(gofumpt -d -e . | tee /dev/stderr)"

lint:
	@$(set_env) ./.github/pre_install.sh
	@$(set_env) go fmt ./...
	@$(set_env) remove_import_blanklines
	@$(set_env) goimports -local $(local_package) -w .
	@$(set_env) gofumpt -l -w .

build:
	go build ./...

test:
	./.github/test.sh

html:
	go tool cover -html=c.out
