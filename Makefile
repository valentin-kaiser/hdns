build: prod

.PHONY: build prod repair lint schema

prod: 
	$(MAKE) repair -C frontend
	$(MAKE) prod -C frontend
	$(MAKE) repair -C backend
	$(MAKE) prod -C backend
	$(MAKE) prod -C worker

repair:
	$(MAKE) repair -C backend/
	$(MAKE) repair -C frontend/
	$(MAKE) repair -C worker/

lint:
	$(MAKE) lint -C backend/
	$(MAKE) lint -C worker/

schema:
	$(MAKE) schema -C backend/pkg/database/
