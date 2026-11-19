.PHONY: test
TESTS_DIR := tests

test:
	@if [ -z "$(filter-out $@,$(MAKECMDGOALS))" ]; then \
		echo "Usage: make test <name>"; \
		echo "Available tests:"; \
		find $(TESTS_DIR) -maxdepth 1 -name '*.js' -exec basename {} .js \; 2>/dev/null; \
		exit 0; \
		fi
	@for test in $(filter-out $@,$(MAKECMDGOALS)); do \
		if [ ! -f "$(TESTS_DIR)/$$test.js" ]; then \
		echo "Error: tests/$$test.js does not exist"; \
		exit 1; \
		fi; \
		docker compose run --rm k6 run /scripts/$$test.js || exit $$?; \
		done

%:
	@:
