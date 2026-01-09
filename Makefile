build: prod

.PHONY: build prod repair lint schema

prod: 
	$(MAKE) repair -C application/frontend
	$(MAKE) prod -C application/frontend
	$(MAKE) repair -C application/backend
	$(MAKE) prod -C application/backend

repair:
	$(MAKE) repair -C application/backend/
	$(MAKE) repair -C application/frontend/

lint:
	$(MAKE) lint -C application/backend/

schema:
	$(MAKE) schema -C application/backend/pkg/database/
