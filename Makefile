# Lexora / MindLaw — command deploy & admin akun ke server prod.
# Butuh SSH key di ~/.ssh/mindlaw_deploy (BatchMode, tanpa password).

SSH    := ssh -i "$(HOME)/.ssh/mindlaw_deploy" -o BatchMode=yes root@103.193.179.9
REMOTE := /root/lexora
DC     := docker compose -f docker-compose.prod.yml

# file yang JANGAN ikut ke-sync (secret lokal / artefak build)
EXCLUDES := --exclude=.git --exclude=node_modules --exclude=frontend/node_modules \
            --exclude=frontend/.next --exclude=.next --exclude=.env \
            --exclude=.env.deploy --exclude=.env.example --exclude=server.log --exclude=server.err

.PHONY: help sync deploy deploy-backend deploy-frontend ps logs restart shell \
        user-list user-create user-passwd user-activate user-deactivate user-delete

help:
	@echo "Deploy:"
	@echo "  make deploy            sync kode + build + migrate + seed + up (full)"
	@echo "  make deploy-backend    sync + rebuild+restart backend saja"
	@echo "  make deploy-frontend   sync + rebuild+restart frontend saja"
	@echo "Ops:"
	@echo "  make ps                status container"
	@echo "  make logs [S=backend]  ikuti log (default semua)"
	@echo "  make restart [S=..]    restart service"
	@echo "  make shell             SSH ke server"
	@echo "Akun (CRUD):"
	@echo "  make user-list"
	@echo "  make user-create EMAIL=a@b.com PASS=xxx NAME='Nama' [ROLE=none|super_admin]"
	@echo "  make user-passwd EMAIL=a@b.com PASS=baru"
	@echo "  make user-activate EMAIL=a@b.com"
	@echo "  make user-deactivate EMAIL=a@b.com"
	@echo "  make user-delete EMAIL=a@b.com"

# --- deploy ---
sync:
	tar czf - $(EXCLUDES) . | $(SSH) 'cd $(REMOTE) && tar xzf -'

deploy: sync
	$(SSH) 'cd $(REMOTE) && ./deploy.sh'

deploy-backend: sync
	$(SSH) 'cd $(REMOTE) && $(DC) up -d --build backend'

deploy-frontend: sync
	$(SSH) 'cd $(REMOTE) && $(DC) up -d --build frontend'

# --- ops ---
ps:
	$(SSH) 'cd $(REMOTE) && $(DC) ps'

logs:
	$(SSH) 'cd $(REMOTE) && $(DC) logs -f --tail 100 $(S)'

restart:
	$(SSH) 'cd $(REMOTE) && $(DC) restart $(S)'

shell:
	$(SSH)

# --- akun (usertool jalan di dalam container backend) ---
USERTOOL = $(DC) run --rm backend ./usertool

user-list:
	$(SSH) 'cd $(REMOTE) && $(USERTOOL) list'

user-create:
	$(SSH) 'cd $(REMOTE) && $(USERTOOL) create "$(EMAIL)" "$(PASS)" "$(NAME)" $(ROLE)'

user-passwd:
	$(SSH) 'cd $(REMOTE) && $(USERTOOL) passwd "$(EMAIL)" "$(PASS)"'

user-activate:
	$(SSH) 'cd $(REMOTE) && $(USERTOOL) activate "$(EMAIL)"'

user-deactivate:
	$(SSH) 'cd $(REMOTE) && $(USERTOOL) deactivate "$(EMAIL)"'

user-delete:
	$(SSH) 'cd $(REMOTE) && $(USERTOOL) delete "$(EMAIL)"'
