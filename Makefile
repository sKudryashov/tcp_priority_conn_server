build:
	docker-compose build --no-cache

up:
	docker-compose -f docker-compose.yml up --force-recreate

takeoff:
	$(MAKE) build && $(MAKE) up

stack-test:
	$(RUBY) ./stack-test.rb

stack-single-request:
	$(RUBY) ./stack-test.rb -n test_single_request

test-conn-evicted:
	go test ./... 
