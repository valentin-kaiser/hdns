build: prod

.PHONY: build prod repair lint schema

prod: 
	$(MAKE) repair -C frontend
	$(MAKE) prod -C frontend
	$(MAKE) repair -C backend
	$(MAKE) prod -C backend

repair:
	$(MAKE) repair -C backend/
	$(MAKE) repair -C frontend/

lint:
	$(MAKE) lint -C backend/

schema:
	$(MAKE) schema -C backend/pkg/database/
