.PHONY: run test build-local build-lambda clean

run:
	@$(MAKE) -C app run

test:
	@$(MAKE) -C app test

build-local:
	@$(MAKE) -C app build-local

build-lambda:
	@$(MAKE) -C app build-lambda

clean:
	@$(MAKE) -C app clean
